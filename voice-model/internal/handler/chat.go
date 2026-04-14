package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"voice-model/internal/conversation"
	"voice-model/internal/provider"
)

type ChatHandler struct {
	store    *conversation.Store
	registry *provider.Registry
	cfg      ChatConfig
}

type ChatConfig struct {
	SystemPrompt string
	MaxMessages  int
}

func NewChatHandler(store *conversation.Store, reg *provider.Registry, cfg ChatConfig) *ChatHandler {
	return &ChatHandler{store: store, registry: reg, cfg: cfg}
}

type chatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversation_id"`
}

type chatResponse struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	Content        string `json:"content"`
	InputMode      string `json:"input_mode"`
	Timestamp      string `json:"timestamp"`
}

func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	conv := h.getOrCreateConversation(req.ConversationID)
	userMsg := conv.AddMessage("user", req.Message, "text")
	msgs := conv.GetMessagesForProvider()

	result, err := h.registry.SendMessage(r.Context(), msgs, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI service unavailable, please retry")
		return
	}

	assistantMsg := conv.AddMessage("assistant", result, "text")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse{
		ConversationID: conv.ID,
		MessageID:      assistantMsg.ID,
		Content:        result,
		InputMode:      "text",
		Timestamp:      userMsg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *ChatHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	conv := h.getOrCreateConversation(req.ConversationID)
	conv.AddMessage("user", req.Message, "text")
	msgs := conv.GetMessagesForProvider()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	result, err := h.registry.SendMessage(r.Context(), msgs, func(chunk string) {
		data, _ := json.Marshal(map[string]string{"type": "chunk", "content": chunk})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	})

	if err != nil {
		data, _ := json.Marshal(map[string]string{"type": "error", "error": "AI service unavailable"})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	assistantMsg := conv.AddMessage("assistant", result, "text")
	data, _ := json.Marshal(map[string]string{
		"type":            "done",
		"message_id":      assistantMsg.ID,
		"conversation_id": conv.ID,
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func (h *ChatHandler) getOrCreateConversation(id string) *conversation.Conversation {
	if id != "" {
		if conv, ok := h.store.Get(id); ok {
			return conv
		}
	}
	return h.store.Create(h.registry.ActiveName(), h.cfg.SystemPrompt, h.cfg.MaxMessages)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
