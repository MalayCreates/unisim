package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/usip/backend/internal/registry"
	"github.com/usip/backend/internal/store"
)

// Runner is the orchestrator capability the API needs: hand off a created run
// for asynchronous execution. Defined here to keep the API decoupled from the
// concrete orchestrator implementation.
type Runner interface {
	Submit(runID string)
}

// NewRouter wires all REST endpoints under /api/v1 and returns the handler.
func NewRouter(st store.Store, reg *registry.Registry, runner Runner) http.Handler {
	r := mux.NewRouter()
	r.Use(corsMiddleware)

	v1 := r.PathPrefix("/api/v1").Subrouter()

	scenarios := &scenarioHandler{st: st}
	v1.HandleFunc("/scenarios", scenarios.list).Methods(http.MethodGet)
	v1.HandleFunc("/scenarios", scenarios.create).Methods(http.MethodPost)
	v1.HandleFunc("/scenarios/{id}", scenarios.get).Methods(http.MethodGet)
	v1.HandleFunc("/scenarios/{id}", scenarios.update).Methods(http.MethodPut)
	v1.HandleFunc("/scenarios/{id}", scenarios.delete).Methods(http.MethodDelete)

	runs := &runHandler{st: st, runner: runner}
	v1.HandleFunc("/scenarios/{id}/runs", runs.list).Methods(http.MethodGet)
	v1.HandleFunc("/scenarios/{id}/runs", runs.trigger).Methods(http.MethodPost)
	v1.HandleFunc("/runs/{runID}", runs.get).Methods(http.MethodGet)
	v1.HandleFunc("/runs/{runID}/results", runs.getResults).Methods(http.MethodGet)

	batches := &batchHandler{st: st, runner: runner}
	v1.HandleFunc("/scenarios/{id}/batches", batches.create).Methods(http.MethodPost)
	v1.HandleFunc("/batches/{batchID}", batches.get).Methods(http.MethodGet)

	adapters := &adapterHandler{reg: reg}
	v1.HandleFunc("/adapters", adapters.register).Methods(http.MethodPost)
	v1.HandleFunc("/adapters", adapters.list).Methods(http.MethodGet)

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}).Methods(http.MethodGet)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
