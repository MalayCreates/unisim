// Package runner brings sim-engine adapters online on demand. When the dispatch
// queue has a run for an engine that has no live adapter, it asks the runner
// manager to launch one; the adapter then self-registers and the run proceeds.
//
// v1 supports the "process" runner type (exec a local adapter binary or script).
// The Recipe shape leaves room for a "docker" type — and, later, remote runners
// that register their gRPC address over HTTP — without changing the manager's
// interface.
package runner

import (
	"encoding/json"
	"fmt"
	"os"
)

// RunnerType selects how an adapter is launched.
type RunnerType string

const (
	// TypeProcess execs a local command (the v1 default).
	TypeProcess RunnerType = "process"
	// TypeDocker is reserved: `docker run` a containerized adapter. Not yet
	// implemented — recognized so configs can be authored ahead of support.
	TypeDocker RunnerType = "docker"
)

// Recipe describes how to launch the adapter for one engine.
type Recipe struct {
	Type    RunnerType        `json:"type"`
	Cmd     string            `json:"cmd"`
	Args    []string          `json:"args"`
	Workdir string            `json:"workdir"`
	Env     map[string]string `json:"env"`
}

// Config is the on-disk runner_config.json: a map of engine ID to launch recipe.
type Config struct {
	Runners map[string]Recipe `json:"runners"`
}

// LoadConfig reads runner_config.json from path. A missing file is not an error
// — it yields (nil, nil) so the caller can run with spawn-on-demand disabled.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read runner config %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse runner config %s: %w", path, err)
	}
	return &c, nil
}
