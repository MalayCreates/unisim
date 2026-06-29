package api

import (
	"encoding/json"
	"net/http"

	"github.com/usip/backend/internal/registry"
)

type adapterHandler struct {
	reg *registry.Registry
}

type registerAdapterRequest struct {
	EngineID string `json:"engine_id"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Version  string `json:"version"`
}

// register is the endpoint adapters POST to on startup to announce themselves.
func (h *adapterHandler) register(w http.ResponseWriter, r *http.Request) {
	var req registerAdapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.EngineID == "" || req.Host == "" || req.Port == 0 {
		writeError(w, http.StatusBadRequest, "engine_id, host, and port are required")
		return
	}
	a := registry.Adapter{
		EngineID: req.EngineID,
		Host:     req.Host,
		Port:     req.Port,
		Version:  req.Version,
	}
	h.reg.Register(a)
	writeJSON(w, http.StatusCreated, h.mustGet(req.EngineID))
}

func (h *adapterHandler) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.reg.List())
}

func (h *adapterHandler) mustGet(engineID string) registry.Adapter {
	a, _ := h.reg.Get(engineID)
	return a
}
