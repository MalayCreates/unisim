package store

import "context"

// Store is the persistence interface. SQLite backs it in dev; Postgres can be
// dropped in later without touching any business logic.
type Store interface {
	// Scenario CRUD
	CreateScenario(ctx context.Context, s *Scenario) error
	GetScenario(ctx context.Context, id string) (*Scenario, error)
	ListScenarios(ctx context.Context) ([]*Scenario, error)
	UpdateScenario(ctx context.Context, s *Scenario) error
	DeleteScenario(ctx context.Context, id string) error

	// Run lifecycle
	CreateRun(ctx context.Context, r *Run) error
	GetRun(ctx context.Context, id string) (*Run, error)
	ListRunsForScenario(ctx context.Context, scenarioID string) ([]*Run, error)
	UpdateRunStatus(ctx context.Context, id string, status RunStatus, errMsg string) error

	// Dispatch queue
	// ListQueuedRuns returns up to limit pending (queued) runs, highest
	// priority first then oldest first — the order the dispatcher should
	// consider them in.
	ListQueuedRuns(ctx context.Context, limit int) ([]*Run, error)
	// ClaimRun atomically transitions a run from pending to running and stamps
	// the claiming worker. It returns true only if this caller won the claim.
	ClaimRun(ctx context.Context, id, workerID string) (bool, error)
	// RequeueRunningRuns resets any runs stuck in 'running' back to pending.
	// Called on startup to recover runs orphaned by a crash/restart. Returns
	// the number of runs requeued.
	RequeueRunningRuns(ctx context.Context) (int, error)

	// Results
	SaveResults(ctx context.Context, r *SimResults) error
	GetResults(ctx context.Context, runID string) (*SimResults, error)
}
