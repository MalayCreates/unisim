package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/usip/backend/internal/store"
)

type runHandler struct {
	st store.Store
}

type triggerRunRequest struct {
	EngineID string `json:"engine_id"`
}

// triggerRun creates a new Run record and returns it; the orchestrator will
// pick it up and execute it asynchronously (wired in the orchestrator package).
func (h *runHandler) trigger(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["id"]
	if _, err := h.st.GetScenario(r.Context(), scenarioID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "scenario not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req triggerRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.EngineID == "" {
		req.EngineID = "custom-engine" // default to built-in adapter
	}

	now := time.Now().UTC()
	run := &store.Run{
		ID:         uuid.NewString(),
		ScenarioID: scenarioID,
		EngineID:   req.EngineID,
		Status:     store.RunStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.st.CreateRun(r.Context(), run); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (h *runHandler) get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["runID"]
	run, err := h.st.GetRun(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *runHandler) list(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["id"]
	runs, err := h.st.ListRunsForScenario(r.Context(), scenarioID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if runs == nil {
		runs = []*store.Run{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (h *runHandler) getResults(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["runID"]
	results, err := h.st.GetResults(r.Context(), runID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "results not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}
