package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/usip/backend/internal/store"
)

// maxBatchReplications caps how many runs a single batch request can create,
// to keep an accidental large request from flooding the queue.
const maxBatchReplications = 50

type batchHandler struct {
	st     store.Store
	runner Runner
}

type createBatchRequest struct {
	EngineID string `json:"engine_id"`
	Count    int    `json:"count"`
}

// BatchMOEAggregate summarizes one MOE key across all completed runs in a
// batch — the point of running N replications of a stochastic scenario.
type BatchMOEAggregate struct {
	Key    string  `json:"key"`
	Unit   string  `json:"unit"`
	Count  int     `json:"count"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stddev"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

// BatchSummary is the aggregate view of a batch: per-run status counts plus
// cross-run MOE statistics computed from whichever runs have completed so
// far (it's valid to poll this before every run finishes).
type BatchSummary struct {
	BatchID        string              `json:"batch_id"`
	ScenarioID     string              `json:"scenario_id"`
	EngineID       string              `json:"engine_id"`
	Total          int                 `json:"total"`
	Pending        int                 `json:"pending"`
	Running        int                 `json:"running"`
	Completed      int                 `json:"completed"`
	Failed         int                 `json:"failed"`
	Runs           []*store.Run        `json:"runs"`
	AggregatedMOEs []BatchMOEAggregate `json:"aggregated_moes"`
}

// create kicks off count independent runs of the same scenario (Monte Carlo
// replications), tagged with a shared batch ID. Each run gets its own run ID
// so the engine's per-run-ID RNG seeding still produces an independent
// outcome per replication.
func (h *batchHandler) create(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["id"]
	if _, err := h.st.GetScenario(r.Context(), scenarioID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "scenario not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req createBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.EngineID == "" {
		req.EngineID = "custom-engine"
	}
	if req.Count < 1 || req.Count > maxBatchReplications {
		writeError(w, http.StatusBadRequest, "count must be between 1 and 50")
		return
	}

	batchID := uuid.NewString()
	now := time.Now().UTC()
	runIDs := make([]string, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		run := &store.Run{
			ID:         uuid.NewString(),
			ScenarioID: scenarioID,
			EngineID:   req.EngineID,
			Status:     store.RunStatusPending,
			BatchID:    batchID,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := h.st.CreateRun(r.Context(), run); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		runIDs = append(runIDs, run.ID)
		if h.runner != nil {
			h.runner.Submit(run.ID)
		}
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"batch_id": batchID,
		"run_ids":  runIDs,
	})
}

func (h *batchHandler) get(w http.ResponseWriter, r *http.Request) {
	batchID := mux.Vars(r)["batchID"]
	runs, err := h.st.ListRunsForBatch(r.Context(), batchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(runs) == 0 {
		writeError(w, http.StatusNotFound, "batch not found")
		return
	}

	summary := BatchSummary{
		BatchID:    batchID,
		ScenarioID: runs[0].ScenarioID,
		EngineID:   runs[0].EngineID,
		Total:      len(runs),
		Runs:       runs,
		// Non-nil so it serializes as [] rather than null before any
		// replication completes; the frontend indexes .length on it.
		AggregatedMOEs: []BatchMOEAggregate{},
	}

	// key -> unit, collected values across completed runs.
	valuesByKey := map[string][]float64{}
	unitByKey := map[string]string{}

	for _, run := range runs {
		switch run.Status {
		case store.RunStatusPending:
			summary.Pending++
		case store.RunStatusRunning:
			summary.Running++
		case store.RunStatusFailed:
			summary.Failed++
		case store.RunStatusCompleted:
			summary.Completed++
			res, err := h.st.GetResults(r.Context(), run.ID)
			if err != nil {
				continue // shouldn't happen for a completed run, but don't fail the whole summary
			}
			for _, m := range res.MOEMetrics {
				valuesByKey[m.Key] = append(valuesByKey[m.Key], m.Value)
				unitByKey[m.Key] = m.Unit
			}
		}
	}

	for key, values := range valuesByKey {
		summary.AggregatedMOEs = append(summary.AggregatedMOEs, aggregateMOE(key, unitByKey[key], values))
	}
	sort.Slice(summary.AggregatedMOEs, func(i, j int) bool {
		return summary.AggregatedMOEs[i].Key < summary.AggregatedMOEs[j].Key
	})

	writeJSON(w, http.StatusOK, summary)
}

// aggregateMOE computes descriptive statistics for one MOE key across a
// batch's completed runs. Uses sample standard deviation (n-1) since these
// runs are a sample of the scenario's possible stochastic outcomes.
func aggregateMOE(key, unit string, values []float64) BatchMOEAggregate {
	agg := BatchMOEAggregate{Key: key, Unit: unit, Count: len(values)}
	if len(values) == 0 {
		return agg
	}
	agg.Min, agg.Max = values[0], values[0]
	var sum float64
	for _, v := range values {
		sum += v
		if v < agg.Min {
			agg.Min = v
		}
		if v > agg.Max {
			agg.Max = v
		}
	}
	agg.Mean = sum / float64(len(values))

	if len(values) > 1 {
		var sqDiff float64
		for _, v := range values {
			d := v - agg.Mean
			sqDiff += d * d
		}
		agg.StdDev = math.Sqrt(sqDiff / float64(len(values)-1))
	}
	return agg
}
