// Package queue turns run submission into an asynchronous, persisted dispatch
// queue. Runs are created as pending in the store; a single dispatcher
// goroutine claims pending runs (highest priority, then oldest first) and hands
// them to a pool of worker goroutines that drive the actual simulation via the
// Executor. Multiple runs — including ones targeting different engines — execute
// in parallel up to the worker count.
//
// The queue is SQLite-backed: claim is an atomic conditional UPDATE, and runs
// left in 'running' by a crash are requeued on startup. This keeps the door
// open for multiple backend instances sharing one database later.
package queue

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/usip/backend/internal/store"
)

// Executor runs a single simulation end to end. *orchestrator.Orchestrator
// satisfies this.
type Executor interface {
	Execute(ctx context.Context, runID string) error
}

// Registry reports whether an adapter is available for an engine.
type Registry interface {
	Has(engineID string) bool
}

// RunnerManager can bring an adapter online on demand. Optional — when nil, the
// dispatcher simply waits for an adapter to register (or times the run out).
type RunnerManager interface {
	EnsureRunning(engineID string) error
}

// Config tunes the dispatcher. Zero values fall back to sensible defaults.
type Config struct {
	Workers      int           // max concurrent runs (default 4)
	PollInterval time.Duration // fallback poll cadence (default 1s)
	MaxQueueWait time.Duration // give up if no adapter appears in time (default 2m)
	WorkerID     string        // stamped on claimed runs (default "usip-dispatcher")
}

func (c *Config) withDefaults() {
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.PollInterval <= 0 {
		c.PollInterval = time.Second
	}
	if c.MaxQueueWait <= 0 {
		c.MaxQueueWait = 2 * time.Minute
	}
	if c.WorkerID == "" {
		c.WorkerID = "usip-dispatcher"
	}
}

// Dispatcher claims queued runs and executes them on a worker pool.
type Dispatcher struct {
	st     store.Store
	reg    Registry
	exec   Executor
	runner RunnerManager // may be nil
	cfg    Config

	jobs     chan string
	notify   chan struct{}
	inflight int32 // runs claimed but not yet finished
}

// New builds a dispatcher. runner may be nil to disable spawn-on-demand.
func New(st store.Store, reg Registry, exec Executor, runner RunnerManager, cfg Config) *Dispatcher {
	cfg.withDefaults()
	return &Dispatcher{
		st:     st,
		reg:    reg,
		exec:   exec,
		runner: runner,
		cfg:    cfg,
		jobs:   make(chan string, cfg.Workers),
		notify: make(chan struct{}, 1),
	}
}

// Submit satisfies the api.Runner interface. The run is already persisted as
// pending by the handler; this just nudges the dispatcher to look sooner.
func (d *Dispatcher) Submit(runID string) {
	select {
	case d.notify <- struct{}{}:
	default: // a poll is already pending; nothing to do
	}
}

// Start launches the worker pool and the dispatch loop. It returns immediately;
// both stop when ctx is cancelled. Any runs orphaned in 'running' by a previous
// crash are requeued first.
func (d *Dispatcher) Start(ctx context.Context) {
	if n, err := d.st.RequeueRunningRuns(ctx); err != nil {
		log.Printf("queue: requeue on startup failed: %v", err)
	} else if n > 0 {
		log.Printf("queue: requeued %d interrupted run(s)", n)
	}

	for i := 0; i < d.cfg.Workers; i++ {
		go d.worker(ctx)
	}
	go d.loop(ctx)
	log.Printf("queue: dispatcher started (workers=%d, poll=%s)", d.cfg.Workers, d.cfg.PollInterval)
}

func (d *Dispatcher) loop(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.PollInterval)
	defer ticker.Stop()
	for {
		d.pollOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-d.notify:
		}
	}
}

// pollOnce claims and dispatches as many ready runs as there is spare worker
// capacity for.
func (d *Dispatcher) pollOnce(ctx context.Context) {
	capacity := d.cfg.Workers - int(atomic.LoadInt32(&d.inflight))
	if capacity <= 0 {
		return
	}
	runs, err := d.st.ListQueuedRuns(ctx, capacity)
	if err != nil {
		log.Printf("queue: list queued runs: %v", err)
		return
	}
	for _, run := range runs {
		if capacity <= 0 {
			break
		}
		if !d.reg.Has(run.EngineID) {
			d.handleMissingAdapter(ctx, run)
			continue
		}
		claimed, err := d.st.ClaimRun(ctx, run.ID, d.cfg.WorkerID)
		if err != nil {
			log.Printf("queue: claim run %s: %v", run.ID, err)
			continue
		}
		if !claimed {
			continue // someone else got it
		}
		atomic.AddInt32(&d.inflight, 1)
		d.jobs <- run.ID
		capacity--
	}
}

// handleMissingAdapter asks the runner manager (if any) to bring the engine
// online, and fails runs that have waited too long for an adapter.
func (d *Dispatcher) handleMissingAdapter(ctx context.Context, run *store.Run) {
	if d.runner != nil {
		if err := d.runner.EnsureRunning(run.EngineID); err != nil {
			log.Printf("queue: ensure %q running: %v", run.EngineID, err)
		}
	}
	if time.Since(run.CreatedAt) > d.cfg.MaxQueueWait {
		msg := "no adapter available for engine " + run.EngineID
		if err := d.st.UpdateRunStatus(ctx, run.ID, store.RunStatusFailed, msg); err != nil {
			log.Printf("queue: fail stale run %s: %v", run.ID, err)
		} else {
			log.Printf("queue: run %s failed — %s (waited %s)", run.ID, msg, d.cfg.MaxQueueWait)
		}
	}
}

func (d *Dispatcher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case runID := <-d.jobs:
			d.runOne(ctx, runID)
			atomic.AddInt32(&d.inflight, -1)
			d.Submit("") // capacity freed; poll again promptly
		}
	}
}

func (d *Dispatcher) runOne(ctx context.Context, runID string) {
	if err := d.exec.Execute(ctx, runID); err != nil {
		log.Printf("queue: run %s failed: %v", runID, err)
		if uerr := d.st.UpdateRunStatus(ctx, runID, store.RunStatusFailed, err.Error()); uerr != nil {
			log.Printf("queue: mark run %s failed: %v", runID, uerr)
		}
	}
}
