package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS scenarios (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	data        TEXT NOT NULL,  -- JSON blob of full Scenario struct
	created_at  DATETIME NOT NULL,
	updated_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
	id          TEXT PRIMARY KEY,
	scenario_id TEXT NOT NULL REFERENCES scenarios(id),
	engine_id   TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'pending',
	error       TEXT NOT NULL DEFAULT '',
	priority    INTEGER NOT NULL DEFAULT 0,
	worker_id   TEXT NOT NULL DEFAULT '',
	claimed_at  DATETIME,
	created_at  DATETIME NOT NULL,
	updated_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS results (
	id          TEXT PRIMARY KEY,
	run_id      TEXT NOT NULL REFERENCES runs(id),
	scenario_id TEXT NOT NULL,
	engine_id   TEXT NOT NULL,
	data        TEXT NOT NULL,  -- JSON blob of SimResults struct
	created_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runs_scenario ON runs(scenario_id);
CREATE INDEX IF NOT EXISTS idx_results_run   ON results(run_id);
`

type sqliteStore struct {
	db *sql.DB
}

// NewSQLite opens (or creates) the SQLite file at path and runs migrations.
func NewSQLite(path string) (Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}
	if err := migrateRuns(db); err != nil {
		return nil, fmt.Errorf("migrate runs: %w", err)
	}
	return &sqliteStore{db: db}, nil
}

// migrateRuns adds queue columns to pre-existing runs tables. SQLite has no
// "ADD COLUMN IF NOT EXISTS", so we run each ALTER and tolerate the
// duplicate-column error on databases that already have them.
func migrateRuns(db *sql.DB) error {
	alters := []string{
		`ALTER TABLE runs ADD COLUMN priority INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE runs ADD COLUMN worker_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runs ADD COLUMN claimed_at DATETIME`,
	}
	for _, stmt := range alters {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}

// --- Scenario CRUD ---

func (s *sqliteStore) CreateScenario(ctx context.Context, sc *Scenario) error {
	blob, err := json.Marshal(sc)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO scenarios (id, name, description, data, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sc.ID, sc.Name, sc.Description, string(blob), sc.CreatedAt, sc.UpdatedAt,
	)
	return err
}

func (s *sqliteStore) GetScenario(ctx context.Context, id string) (*Scenario, error) {
	row := s.db.QueryRowContext(ctx, `SELECT data FROM scenarios WHERE id = ?`, id)
	var blob string
	if err := row.Scan(&blob); err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	var sc Scenario
	return &sc, json.Unmarshal([]byte(blob), &sc)
}

func (s *sqliteStore) ListScenarios(ctx context.Context) ([]*Scenario, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM scenarios ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Scenario
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		var sc Scenario
		if err := json.Unmarshal([]byte(blob), &sc); err != nil {
			return nil, err
		}
		out = append(out, &sc)
	}
	return out, rows.Err()
}

func (s *sqliteStore) UpdateScenario(ctx context.Context, sc *Scenario) error {
	sc.UpdatedAt = time.Now().UTC()
	blob, err := json.Marshal(sc)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE scenarios SET name = ?, description = ?, data = ?, updated_at = ? WHERE id = ?`,
		sc.Name, sc.Description, string(blob), sc.UpdatedAt, sc.ID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteStore) DeleteScenario(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM scenarios WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Runs ---

func (s *sqliteStore) CreateRun(ctx context.Context, r *Run) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO runs (id, scenario_id, engine_id, status, error, priority, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ScenarioID, r.EngineID, string(r.Status), r.Error, r.Priority, r.CreatedAt, r.UpdatedAt,
	)
	return err
}

const runColumns = `id, scenario_id, engine_id, status, error, priority, worker_id, claimed_at, created_at, updated_at`

func (s *sqliteStore) GetRun(ctx context.Context, id string) (*Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+runColumns+` FROM runs WHERE id = ?`, id)
	return scanRun(row)
}

func (s *sqliteStore) ListRunsForScenario(ctx context.Context, scenarioID string) ([]*Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+runColumns+` FROM runs WHERE scenario_id = ? ORDER BY created_at DESC`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListQueuedRuns returns pending runs in dispatch order: highest priority
// first, then oldest first.
func (s *sqliteStore) ListQueuedRuns(ctx context.Context, limit int) ([]*Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+runColumns+` FROM runs WHERE status = ?
		 ORDER BY priority DESC, created_at ASC LIMIT ?`,
		string(RunStatusPending), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClaimRun atomically moves a run from pending to running. The WHERE guard on
// status makes the claim safe against concurrent dispatchers: only one UPDATE
// can match, so only one caller sees rowsAffected == 1.
func (s *sqliteStore) ClaimRun(ctx context.Context, id, workerID string) (bool, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, worker_id = ?, claimed_at = ?, updated_at = ?
		 WHERE id = ? AND status = ?`,
		string(RunStatusRunning), workerID, now, now, id, string(RunStatusPending),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// RequeueRunningRuns resets interrupted runs (status running) back to pending
// so the dispatcher re-runs them after a restart.
func (s *sqliteStore) RequeueRunningRuns(ctx context.Context) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, worker_id = '', claimed_at = NULL, updated_at = ?
		 WHERE status = ?`,
		string(RunStatusPending), time.Now().UTC(), string(RunStatusRunning),
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *sqliteStore) UpdateRunStatus(ctx context.Context, id string, status RunStatus, errMsg string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, error = ?, updated_at = ? WHERE id = ?`,
		string(status), errMsg, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Results ---

func (s *sqliteStore) SaveResults(ctx context.Context, r *SimResults) error {
	blob, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO results (id, run_id, scenario_id, engine_id, data, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.RunID, r.ScenarioID, r.EngineID, string(blob), r.CreatedAt,
	)
	return err
}

func (s *sqliteStore) GetResults(ctx context.Context, runID string) (*SimResults, error) {
	row := s.db.QueryRowContext(ctx, `SELECT data FROM results WHERE run_id = ?`, runID)
	var blob string
	if err := row.Scan(&blob); err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	var r SimResults
	return &r, json.Unmarshal([]byte(blob), &r)
}

// --- helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanRun(s scanner) (*Run, error) {
	var r Run
	var status string
	var claimedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := s.Scan(&r.ID, &r.ScenarioID, &r.EngineID, &status, &r.Error,
		&r.Priority, &r.WorkerID, &claimedAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r.Status = RunStatus(status)
	if claimedAt.Valid {
		r.ClaimedAt = &claimedAt.Time
	}
	r.CreatedAt = createdAt
	r.UpdatedAt = updatedAt
	return &r, nil
}
