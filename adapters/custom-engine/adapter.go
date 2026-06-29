package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/usip/backend/schema"
)

const (
	adapterVersion = "custom-engine/0.1.0"
	engineID       = "custom-engine"
)

// runState tracks one in-flight or completed simulation run.
type runState struct {
	done    chan struct{}
	results *schema.SimResultsProto
	err     error
}

// adapterServer implements the SimAdapter gRPC service.
type adapterServer struct {
	schema.UnimplementedSimAdapterServer // forward-compat (no-op with require_unimplemented_servers=false)

	mu        sync.Mutex
	scenarios map[string]*schema.ScenarioProto
	runs      map[string]*runState
}

func newAdapterServer() *adapterServer {
	return &adapterServer{
		scenarios: make(map[string]*schema.ScenarioProto),
		runs:      make(map[string]*runState),
	}
}

func (a *adapterServer) Initialize(ctx context.Context, s *schema.ScenarioProto) (*schema.InitResponse, error) {
	if s.Id == "" {
		return &schema.InitResponse{Success: false, Message: "scenario id is required"}, nil
	}
	a.mu.Lock()
	a.scenarios[s.Id] = s
	a.mu.Unlock()

	log.Printf("Initialize: scenario=%s entities=%d missions=%d", s.Id, len(s.Entities), len(s.Missions))
	return &schema.InitResponse{
		Success:        true,
		Message:        fmt.Sprintf("scenario %q initialized", s.Name),
		AdapterVersion: adapterVersion,
	}, nil
}

func (a *adapterServer) Run(ctx context.Context, req *schema.RunRequest) (*schema.RunResponse, error) {
	a.mu.Lock()
	scenario, ok := a.scenarios[req.ScenarioId]
	if !ok {
		a.mu.Unlock()
		return &schema.RunResponse{Accepted: false, Message: "scenario not initialized; call Initialize first"}, nil
	}
	if _, exists := a.runs[req.RunId]; exists {
		a.mu.Unlock()
		return &schema.RunResponse{Accepted: false, Message: "run already exists"}, nil
	}
	state := &runState{done: make(chan struct{})}
	a.runs[req.RunId] = state
	a.mu.Unlock()

	// Execute asynchronously so Run returns promptly.
	go func() {
		defer close(state.done)
		log.Printf("Run: run=%s scenario=%s starting", req.RunId, req.ScenarioId)
		eng := newEngine(scenario, req.RunId)
		state.results = eng.run(req.ScenarioId, engineID, req.RunId)
		log.Printf("Run: run=%s done — %d tracks, %d events, %d kills",
			req.RunId, len(state.results.EntityTracks), len(state.results.Events), len(state.results.KillChains))
	}()

	return &schema.RunResponse{Accepted: true, Message: "run started"}, nil
}

func (a *adapterServer) GetResults(ctx context.Context, req *schema.ResultsRequest) (*schema.SimResultsProto, error) {
	a.mu.Lock()
	state, ok := a.runs[req.RunId]
	a.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown run %q", req.RunId)
	}

	// Block until the run completes or the caller's context is cancelled.
	select {
	case <-state.done:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if state.err != nil {
		return nil, state.err
	}
	return state.results, nil
}

func (a *adapterServer) Shutdown(ctx context.Context, req *schema.ShutdownRequest) (*schema.ShutdownResponse, error) {
	log.Printf("Shutdown requested (graceful=%v)", req.Graceful)
	return &schema.ShutdownResponse{Success: true}, nil
}
