package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/usip/backend/internal/api"
	"github.com/usip/backend/internal/orchestrator"
	"github.com/usip/backend/internal/queue"
	"github.com/usip/backend/internal/registry"
	"github.com/usip/backend/internal/runner"
	"github.com/usip/backend/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "", "SQLite database path (default: $HOME/.usip/usip.db)")
	publicURL := flag.String("public-url", "", "base URL adapters use to reach this backend (default http://localhost<port>)")
	runnerConfig := flag.String("runner-config", "runner_config.json", "path to runner config (spawn-on-demand); ignored if absent")
	workers := flag.Int("workers", 4, "max concurrent simulation runs")
	flag.Parse()

	if *dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home dir: %v", err)
		}
		dir := filepath.Join(home, ".usip")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("cannot create data dir: %v", err)
		}
		*dbPath = filepath.Join(dir, "usip.db")
	}
	if *publicURL == "" {
		*publicURL = "http://localhost" + portOf(*addr)
	}

	st, err := store.NewSQLite(*dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	reg := registry.New()
	orch := orchestrator.New(st, reg)

	// Runner manager: spawn-on-demand for engines without a live adapter.
	// A missing config file simply disables it (manager stays unconfigured).
	rcfg, err := runner.LoadConfig(*runnerConfig)
	if err != nil {
		log.Fatalf("runner config: %v", err)
	}
	mgr := runner.NewManager(rcfg, *publicURL)
	var rm queue.RunnerManager
	if mgr.Configured() {
		rm = mgr
		log.Printf("runner: spawn-on-demand enabled from %s", *runnerConfig)
	}

	// Dispatch queue: async, persisted, parallel execution.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dispatcher := queue.New(st, reg, orch, rm, queue.Config{Workers: *workers})
	dispatcher.Start(ctx)

	router := api.NewRouter(st, reg, dispatcher)
	srv := &http.Server{Addr: *addr, Handler: router}

	go func() {
		log.Printf("USIP backend listening on %s  db=%s  public=%s", *addr, *dbPath, *publicURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	mgr.Shutdown()
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

// portOf extracts ":8080" -> ":8080" and "0.0.0.0:8080" -> ":8080".
func portOf(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i:]
		}
	}
	return ":8080"
}
