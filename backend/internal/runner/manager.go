package runner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// respawnCooldown throttles relaunch attempts so a crash-looping adapter does
// not get spawned on every dispatcher poll.
const respawnCooldown = 5 * time.Second

// Manager launches and tracks adapter processes per engine. It satisfies the
// queue's RunnerManager interface via EnsureRunning.
type Manager struct {
	recipes    map[string]Recipe
	backendURL string // substituted for the $BACKEND_URL token in recipes

	mu      sync.Mutex
	procs   map[string]*exec.Cmd // engineID -> running process
	lastTry map[string]time.Time // engineID -> last spawn attempt
}

// NewManager builds a manager from config. backendURL replaces the literal
// token "$BACKEND_URL" anywhere in a recipe's args or env, so spawned adapters
// register back to the right place regardless of the configured listen address.
func NewManager(cfg *Config, backendURL string) *Manager {
	recipes := map[string]Recipe{}
	if cfg != nil {
		recipes = cfg.Runners
	}
	return &Manager{
		recipes:    recipes,
		backendURL: backendURL,
		procs:      make(map[string]*exec.Cmd),
		lastTry:    make(map[string]time.Time),
	}
}

// Configured reports whether any runner recipes are defined.
func (m *Manager) Configured() bool { return len(m.recipes) > 0 }

// EnsureRunning launches the adapter for engineID if it is not already running.
// It is idempotent and safe to call on every dispatch poll: a live process or a
// recent spawn attempt short-circuits. The adapter self-registers once up;
// EnsureRunning does not block waiting for that.
func (m *Manager) EnsureRunning(engineID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cmd, ok := m.procs[engineID]; ok && cmd.ProcessState == nil {
		return nil // already running
	}
	if last, ok := m.lastTry[engineID]; ok && time.Since(last) < respawnCooldown {
		return nil // backing off after a recent attempt
	}

	recipe, ok := m.recipes[engineID]
	if !ok {
		return fmt.Errorf("no runner configured for engine %q", engineID)
	}
	if recipe.Type != "" && recipe.Type != TypeProcess {
		return fmt.Errorf("runner type %q for engine %q is not supported yet", recipe.Type, engineID)
	}

	m.lastTry[engineID] = time.Now()
	cmd := m.buildCmd(recipe)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn adapter for %q: %w", engineID, err)
	}
	m.procs[engineID] = cmd
	log.Printf("runner: spawned adapter for %q (pid %d): %s",
		engineID, cmd.Process.Pid, strings.Join(cmd.Args, " "))

	// Reap the process when it exits so it can be respawned on a later run.
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		if m.procs[engineID] == cmd {
			delete(m.procs, engineID)
		}
		m.mu.Unlock()
		log.Printf("runner: adapter for %q exited: %v", engineID, err)
	}()
	return nil
}

func (m *Manager) buildCmd(recipe Recipe) *exec.Cmd {
	args := make([]string, len(recipe.Args))
	for i, a := range recipe.Args {
		args[i] = m.subst(a)
	}
	cmd := exec.Command(recipe.Cmd, args...)
	if recipe.Workdir != "" {
		cmd.Dir = recipe.Workdir
	}
	cmd.Env = os.Environ()
	for k, v := range recipe.Env {
		cmd.Env = append(cmd.Env, k+"="+m.subst(v))
	}
	// Inherit the parent's streams so adapter logs are visible in dev.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (m *Manager) subst(s string) string {
	return strings.ReplaceAll(s, "$BACKEND_URL", m.backendURL)
}

// Shutdown signals all spawned adapter processes to stop. Best-effort.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for engineID, cmd := range m.procs {
		if cmd.Process != nil {
			log.Printf("runner: stopping adapter for %q (pid %d)", engineID, cmd.Process.Pid)
			_ = cmd.Process.Signal(os.Interrupt)
		}
	}
}
