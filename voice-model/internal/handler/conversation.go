package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"voice-model/internal/conversation"
)

type ConversationHandler struct {
	store *conversation.Store
}

func NewConversationHandler(store *conversation.Store) *ConversationHandler {
	return &ConversationHandler{store: store}
}

func (h *ConversationHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conv, ok := h.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}
