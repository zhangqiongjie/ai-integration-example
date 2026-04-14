package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"voice-model/internal/provider"
)

type ProviderHandler struct {
	registry *provider.Registry
}

func NewProviderHandler(reg *provider.Registry) *ProviderHandler {
	return &ProviderHandler{registry: reg}
}

func (h *ProviderHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active":    h.registry.ActiveName(),
		"providers": h.registry.List(),
	})
}

func (h *ProviderHandler) HandleSetActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	if err := h.registry.SetActive(req.Provider); err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("%s provider not configured", req.Provider))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"active":  h.registry.ActiveName(),
		"message": "Provider switched to " + req.Provider,
	})
}
