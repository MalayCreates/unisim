package queue

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/usip/backend/internal/store"
)

// fakeRegistry reports a fixed availability for every engine.
type fakeRegistry struct{ available bool }

func (f fakeRegistry) Has(string) bool { return f.available }

// fakeExecutor records the runs it executed and marks them completed, the way
// the real orchestrator does on success.
type fakeExecutor struct {
	st      store.Store
	mu      sync.Mutex
	seen    []string
	peak    int32
	current int32
}

func (e *fakeExecutor) Execute(ctx context.Context, runID string) error {
	n := atomic.AddInt32(&e.current, 1)
	for { // track peak concurrency
		p := atomic.LoadInt32(&e.peak)
		if n <= p || atomic.CompareAndSwapInt32(&e.peak, p, n) {
			break
		}
	}
	time.Sleep(40 * time.Millisecond)
	atomic.AddInt32(&e.current, -1)

	e.mu.Lock()
	e.seen = append(e.seen, runID)
	e.mu.Unlock()
	return e.st.UpdateRunStatus(ctx, runID, store.RunStatusCompleted, "")
}

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	st, err := store.NewSQLite(filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := st.CreateScenario(context.Background(), &store.Scenario{
		ID: "scn", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("scenario: %v", err)
	}
	return st
}

func enqueueRun(t *testing.T, st store.Store, id string, priority int) {
	t.Helper()
	now := time.Now().UTC()
	if err := st.CreateRun(context.Background(), &store.Run{
		ID: id, ScenarioID: "scn", EngineID: "custom-engine",
		Status: store.RunStatusPending, Priority: priority, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create run %s: %v", id, err)
	}
}

func waitForStatus(t *testing.T, st store.Store, id string, want store.RunStatus, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		r, err := st.GetRun(context.Background(), id)
		if err == nil && r.Status == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	r, _ := st.GetRun(context.Background(), id)
	t.Fatalf("run %s: want status %q, got %q after %s", id, want, r.Status, within)
}

func TestDispatcherRunsAllQueuedInParallel(t *testing.T) {
	st := newTestStore(t)
	exec := &fakeExecutor{st: st}
	d := New(st, fakeRegistry{available: true}, exec, nil, Config{Workers: 3, PollInterval: 5 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	for _, id := range []string{"r1", "r2", "r3", "r4", "r5"} {
		enqueueRun(t, st, id, 0)
		d.Submit(id)
	}
	for _, id := range []string{"r1", "r2", "r3", "r4", "r5"} {
		waitForStatus(t, st, id, store.RunStatusCompleted, 3*time.Second)
	}

	if len(exec.seen) != 5 {
		t.Fatalf("expected 5 runs executed, got %d", len(exec.seen))
	}
	if exec.peak < 2 {
		t.Fatalf("expected parallel execution (peak >= 2), got peak %d", exec.peak)
	}
	if exec.peak > 3 {
		t.Fatalf("peak concurrency %d exceeded worker count 3", exec.peak)
	}
}

func TestDispatcherFailsRunWithNoAdapter(t *testing.T) {
	st := newTestStore(t)
	exec := &fakeExecutor{st: st}
	// No adapter available, no runner manager, tiny wait window.
	d := New(st, fakeRegistry{available: false}, exec, nil,
		Config{Workers: 1, PollInterval: 5 * time.Millisecond, MaxQueueWait: 20 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	enqueueRun(t, st, "orphan", 0)
	d.Submit("orphan")

	waitForStatus(t, st, "orphan", store.RunStatusFailed, 2*time.Second)
	if len(exec.seen) != 0 {
		t.Fatalf("run with no adapter should not execute, but executor saw %v", exec.seen)
	}
}

func TestRequeueRunningRunsOnStartup(t *testing.T) {
	st := newTestStore(t)
	enqueueRun(t, st, "stuck", 0)
	if _, err := st.ClaimRun(context.Background(), "stuck", "old-worker"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	// Simulate a crash: the run is left 'running'. A fresh dispatcher should
	// requeue and complete it.
	exec := &fakeExecutor{st: st}
	d := New(st, fakeRegistry{available: true}, exec, nil, Config{Workers: 1, PollInterval: 5 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	waitForStatus(t, st, "stuck", store.RunStatusCompleted, 2*time.Second)
}

func TestClaimRunIsAtomic(t *testing.T) {
	st := newTestStore(t)
	enqueueRun(t, st, "race", 0)

	const n = 8
	var wins int32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if ok, err := st.ClaimRun(context.Background(), "race", "w"); err == nil && ok {
				atomic.AddInt32(&wins, 1)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("expected exactly one winning claim, got %d", wins)
	}
}
