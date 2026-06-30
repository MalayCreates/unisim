// Package orchestrator manages the simulation run lifecycle: it loads a stored
// scenario, looks up the target adapter in the registry, drives the gRPC
// SimAdapter contract (Initialize -> Run -> GetResults), normalizes the output,
// and persists results while updating run status.
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/usip/backend/internal/normalizer"
	"github.com/usip/backend/internal/registry"
	"github.com/usip/backend/internal/store"
	"github.com/usip/backend/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// runTimeout bounds the entire Initialize->Run->GetResults sequence for a run.
const runTimeout = 5 * time.Minute

type Orchestrator struct {
	st  store.Store
	reg *registry.Registry
}

func New(st store.Store, reg *registry.Registry) *Orchestrator {
	return &Orchestrator{st: st, reg: reg}
}

// Execute drives a single run synchronously through the full adapter contract
// (Initialize -> Run -> GetResults), normalizes the output, and persists it.
// The dispatch queue calls this from a worker goroutine. On success the run is
// marked completed here; on error the caller records the failed status.
func (o *Orchestrator) Execute(ctx context.Context, runID string) error {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	run, err := o.st.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("load run: %w", err)
	}

	adapter, ok := o.reg.Get(run.EngineID)
	if !ok {
		return fmt.Errorf("no adapter registered for engine %q", run.EngineID)
	}

	scenario, err := o.st.GetScenario(ctx, run.ScenarioID)
	if err != nil {
		return fmt.Errorf("load scenario: %w", err)
	}

	if err := o.st.UpdateRunStatus(ctx, runID, store.RunStatusRunning, ""); err != nil {
		return fmt.Errorf("mark running: %w", err)
	}

	// Dial the adapter sidecar.
	conn, err := grpc.NewClient(adapter.Target(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial adapter %s: %w", adapter.Target(), err)
	}
	defer conn.Close()
	client := schema.NewSimAdapterClient(conn)

	// Initialize.
	proto := scenarioToProto(scenario)
	initResp, err := client.Initialize(ctx, proto)
	if err != nil {
		return fmt.Errorf("Initialize: %w", err)
	}
	if !initResp.Success {
		return fmt.Errorf("adapter rejected scenario: %s", initResp.Message)
	}

	// Run.
	runResp, err := client.Run(ctx, &schema.RunRequest{RunId: runID, ScenarioId: scenario.ID})
	if err != nil {
		return fmt.Errorf("Run: %w", err)
	}
	if !runResp.Accepted {
		return fmt.Errorf("adapter rejected run: %s", runResp.Message)
	}

	// GetResults blocks on the adapter side until the run completes.
	results, err := client.GetResults(ctx, &schema.ResultsRequest{RunId: runID})
	if err != nil {
		return fmt.Errorf("GetResults: %w", err)
	}

	// Normalize + persist.
	normalized := normalizer.FromProto(results)
	normalized.RunID = runID
	if err := o.st.SaveResults(ctx, normalized); err != nil {
		return fmt.Errorf("save results: %w", err)
	}
	if err := o.st.UpdateRunStatus(ctx, runID, store.RunStatusCompleted, ""); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	log.Printf("run %s completed via %s: %d tracks, %d events, %d kills",
		runID, run.EngineID, len(normalized.EntityTracks), len(normalized.Events), len(normalized.KillChains))
	return nil
}
