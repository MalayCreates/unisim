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

type scenarioHandler struct {
	st store.Store
}

func (h *scenarioHandler) list(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.st.ListScenarios(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scenarios == nil {
		scenarios = []*store.Scenario{}
	}
	writeJSON(w, http.StatusOK, scenarios)
}

func (h *scenarioHandler) create(w http.ResponseWriter, r *http.Request) {
	var sc store.Scenario
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if sc.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	now := time.Now().UTC()
	sc.ID = uuid.NewString()
	sc.CreatedAt = now
	sc.UpdatedAt = now
	if sc.StartTime.IsZero() {
		sc.StartTime = now
	}
	if err := h.st.CreateScenario(r.Context(), &sc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

func (h *scenarioHandler) get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	sc, err := h.st.GetScenario(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "scenario not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sc)
}

func (h *scenarioHandler) update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	existing, err := h.st.GetScenario(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "scenario not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var patch store.Scenario
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	// Preserve immutable fields.
	patch.ID = existing.ID
	patch.CreatedAt = existing.CreatedAt
	if patch.Name == "" {
		patch.Name = existing.Name
	}
	if err := h.st.UpdateScenario(r.Context(), &patch); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, patch)
}

func (h *scenarioHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.st.DeleteScenario(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "scenario not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
