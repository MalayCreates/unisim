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

	// Results
	SaveResults(ctx context.Context, r *SimResults) error
	GetResults(ctx context.Context, runID string) (*SimResults, error)
}
