package handler

import (
	"encoding/json"
	"net/http"

	"voice-model/internal/provider"
)

type HealthHandler struct {
	registry *provider.Registry
}

func NewHealthHandler(reg *provider.Registry) *HealthHandler {
	return &HealthHandler{registry: reg}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"provider": h.registry.ActiveName(),
		"version":  "1.0.0",
	})
}
