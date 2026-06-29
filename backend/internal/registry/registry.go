// Package registry tracks the sim-engine adapters that have registered
// themselves with the backend. There is no hardcoded engine list — adapters
// announce their host/port on startup and are looked up by engine ID at run time.
package registry

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Adapter describes one registered sim-engine adapter sidecar.
type Adapter struct {
	EngineID     string    `json:"engine_id"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	Version      string    `json:"version"`
	RegisteredAt time.Time `json:"registered_at"`
}

// Target returns the host:port gRPC dial string for this adapter.
func (a Adapter) Target() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

// Registry is an in-memory, concurrency-safe set of registered adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

func New() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

// Register adds or replaces an adapter keyed by its engine ID.
func (r *Registry) Register(a Adapter) {
	a.RegisteredAt = time.Now().UTC()
	r.mu.Lock()
	r.adapters[a.EngineID] = a
	r.mu.Unlock()
}

// Get returns the adapter for engineID, if registered.
func (r *Registry) Get(engineID string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[engineID]
	return a, ok
}

// Deregister removes an adapter (e.g. on graceful shutdown).
func (r *Registry) Deregister(engineID string) {
	r.mu.Lock()
	delete(r.adapters, engineID)
	r.mu.Unlock()
}

// List returns all registered adapters, sorted by engine ID.
func (r *Registry) List() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EngineID < out[j].EngineID })
	return out
}
