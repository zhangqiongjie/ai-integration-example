package provider

import (
	"context"
	"io"
	"time"
)

type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	InputMode string    `json:"input_mode"`
	Timestamp time.Time `json:"timestamp"`
}

type TextProvider interface {
	SendMessage(ctx context.Context, messages []Message, onChunk func(chunk string)) (string, error)
	Name() string
}

type VoiceProvider interface {
	Transcribe(ctx context.Context, audio io.Reader, format string) (string, error)
	Synthesize(ctx context.Context, text string) (io.ReadCloser, error)
}
