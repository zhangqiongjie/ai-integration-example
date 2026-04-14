package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"voice-model/internal/conversation"
	"voice-model/internal/provider"
)

type WSHandler struct {
	store    *conversation.Store
	registry *provider.Registry
	cfg      ChatConfig
}

func NewWSHandler(store *conversation.Store, reg *provider.Registry, cfg ChatConfig) *WSHandler {
	return &WSHandler{store: store, registry: reg, cfg: cfg}
}

type wsMessage struct {
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	Provider string `json:"provider,omitempty"`
	Enabled  *bool  `json:"enabled,omitempty"`
	Format   string `json:"format,omitempty"`
}

type wsConn struct {
	conn         *websocket.Conn
	conv         *conversation.Conversation
	voiceEnabled bool
	audioFormat  string
	audioBuf     bytes.Buffer
	recording    bool
	ttsCancel    context.CancelFunc
	mu           sync.Mutex
}

func (wc *wsConn) writeJSON(msg interface{}) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return wc.conn.Write(context.Background(), websocket.MessageText, data)
}

func (wc *wsConn) writeBinary(data []byte) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	return wc.conn.Write(context.Background(), websocket.MessageBinary, data)
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}
	defer conn.CloseNow()

	convID := r.URL.Query().Get("conversation_id")
	var conv *conversation.Conversation
	if convID != "" {
		conv, _ = h.store.Get(convID)
	}
	if conv == nil {
		conv = h.store.Create(h.registry.ActiveName(), h.cfg.SystemPrompt, h.cfg.MaxMessages)
	}

	wc := &wsConn{
		conn:         conn,
		conv:         conv,
		voiceEnabled: true,
	}

	// Send init message.
	wc.writeJSON(map[string]interface{}{
		"type":            "init",
		"conversation_id": conv.ID,
		"provider":        conv.Provider,
		"voice_enabled":   wc.voiceEnabled,
	})

	ctx := r.Context()
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			log.Printf("ws read error: %v", err)
			return
		}

		if msgType == websocket.MessageBinary {
			// Binary frame = voice audio chunk.
			if wc.recording {
				wc.audioBuf.Write(data)
			}
			continue
		}

		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			wc.writeJSON(map[string]interface{}{
				"type":  "error",
				"error": "invalid message format",
			})
			continue
		}

		switch msg.Type {
		case "text":
			h.handleText(ctx, wc, msg.Content)
		case "voice_start":
			h.handleVoiceStart(wc, msg.Format)
		case "voice_end":
			h.handleVoiceEnd(ctx, wc)
		case "provider_switch":
			h.handleProviderSwitch(wc, msg.Provider)
		case "voice_toggle":
			h.handleVoiceToggle(wc, msg.Enabled)
		default:
			wc.writeJSON(map[string]interface{}{
				"type":  "error",
				"error": "unknown message type: " + msg.Type,
			})
		}
	}
}

func (h *WSHandler) handleText(ctx context.Context, wc *wsConn, content string) {
	if content == "" {
		wc.writeJSON(map[string]interface{}{
			"type":  "error",
			"error": "message is required",
		})
		return
	}

	wc.conv.AddMessage("user", content, "text")
	msgs := wc.conv.GetMessagesForProvider()

	fullContent, err := h.registry.SendMessage(ctx, msgs, func(chunk string) {
		wc.writeJSON(map[string]interface{}{
			"type":    "response_chunk",
			"content": chunk,
		})
	})
	if err != nil {
		wc.writeJSON(map[string]interface{}{
			"type":      "error",
			"error":     "AI service unavailable",
			"retryable": true,
		})
		return
	}

	assistantMsg := wc.conv.AddMessage("assistant", fullContent, "text")
	wc.writeJSON(map[string]interface{}{
		"type":         "response_done",
		"message_id":   assistantMsg.ID,
		"full_content": fullContent,
	})

	// TTS if voice enabled.
	if wc.voiceEnabled {
		h.streamTTS(ctx, wc, fullContent)
	}
}

func (h *WSHandler) handleVoiceStart(wc *wsConn, format string) {
	// Cancel any in-progress TTS.
	if wc.ttsCancel != nil {
		wc.ttsCancel()
	}
	wc.recording = true
	wc.audioFormat = format
	if wc.audioFormat == "" {
		wc.audioFormat = "webm"
	}
	wc.audioBuf.Reset()
}

func (h *WSHandler) handleVoiceEnd(ctx context.Context, wc *wsConn) {
	wc.recording = false
	audioData := wc.audioBuf.Bytes()
	if len(audioData) == 0 {
		wc.writeJSON(map[string]interface{}{
			"type":  "error",
			"error": "no audio data received",
		})
		return
	}

	voice := h.registry.Voice()
	if voice == nil {
		wc.writeJSON(map[string]interface{}{
			"type":  "error",
			"error": "voice provider not available",
		})
		return
	}

	// Transcribe.
	text, err := voice.Transcribe(ctx, bytes.NewReader(audioData), wc.audioFormat)
	if err != nil {
		wc.writeJSON(map[string]interface{}{
			"type":  "error",
			"error": "transcription failed: " + err.Error(),
		})
		return
	}

	if text == "" {
		wc.writeJSON(map[string]interface{}{
			"type":  "error",
			"error": "could not understand audio, please try again",
		})
		return
	}

	userMsg := wc.conv.AddMessage("user", text, "voice")
	wc.writeJSON(map[string]interface{}{
		"type":       "transcription",
		"content":    text,
		"message_id": userMsg.ID,
	})

	// Send transcribed text to AI.
	msgs := wc.conv.GetMessagesForProvider()
	fullContent, err := h.registry.SendMessage(ctx, msgs, func(chunk string) {
		wc.writeJSON(map[string]interface{}{
			"type":    "response_chunk",
			"content": chunk,
		})
	})
	if err != nil {
		wc.writeJSON(map[string]interface{}{
			"type":      "error",
			"error":     "AI service unavailable",
			"retryable": true,
		})
		return
	}

	assistantMsg := wc.conv.AddMessage("assistant", fullContent, "text")
	wc.writeJSON(map[string]interface{}{
		"type":         "response_done",
		"message_id":   assistantMsg.ID,
		"full_content": fullContent,
	})

	// TTS if voice enabled.
	if wc.voiceEnabled {
		h.streamTTS(ctx, wc, fullContent)
	}
}

func (h *WSHandler) handleProviderSwitch(wc *wsConn, name string) {
	if err := h.registry.SetActive(name); err != nil {
		wc.writeJSON(map[string]interface{}{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}
	wc.conv.Provider = name
	wc.writeJSON(map[string]interface{}{
		"type":     "provider_changed",
		"provider": name,
	})
}

func (h *WSHandler) handleVoiceToggle(wc *wsConn, enabled *bool) {
	if enabled != nil {
		wc.voiceEnabled = *enabled
	}
}

func (h *WSHandler) streamTTS(parentCtx context.Context, wc *wsConn, text string) {
	voice := h.registry.Voice()
	if voice == nil {
		return
	}

	ctx, cancel := context.WithCancel(parentCtx)
	wc.ttsCancel = cancel
	defer cancel()

	reader, err := voice.Synthesize(ctx, text)
	if err != nil {
		log.Printf("TTS error: %v", err)
		return
	}
	defer reader.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := reader.Read(buf)
		if n > 0 {
			wc.writeBinary(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("TTS read error: %v", err)
			return
		}
	}

	wc.writeJSON(map[string]interface{}{
		"type": "tts_done",
	})
}
