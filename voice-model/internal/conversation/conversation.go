package conversation

import (
	"time"

	"github.com/google/uuid"
	"voice-model/internal/provider"
)

type Conversation struct {
	ID           string             `json:"conversation_id"`
	Messages     []provider.Message `json:"messages"`
	Provider     string             `json:"provider"`
	CreatedAt    time.Time          `json:"created_at"`
	SystemPrompt string             `json:"-"`
	MaxMessages  int                `json:"-"`
}

func New(providerName, systemPrompt string, maxMessages int) *Conversation {
	return &Conversation{
		ID:           uuid.New().String(),
		Messages:     make([]provider.Message, 0),
		Provider:     providerName,
		CreatedAt:    time.Now(),
		SystemPrompt: systemPrompt,
		MaxMessages:  maxMessages,
	}
}

func (c *Conversation) AddMessage(role, content, inputMode string) provider.Message {
	msg := provider.Message{
		ID:        uuid.New().String(),
		Role:      role,
		Content:   content,
		InputMode: inputMode,
		Timestamp: time.Now(),
	}
	c.Messages = append(c.Messages, msg)
	c.applyWindow()
	return msg
}

func (c *Conversation) GetMessagesForProvider() []provider.Message {
	msgs := make([]provider.Message, 0, len(c.Messages)+1)
	if c.SystemPrompt != "" {
		msgs = append(msgs, provider.Message{
			Role:    "system",
			Content: c.SystemPrompt,
		})
	}
	msgs = append(msgs, c.Messages...)
	return msgs
}

func (c *Conversation) applyWindow() {
	if c.MaxMessages <= 0 || len(c.Messages) <= c.MaxMessages {
		return
	}
	excess := len(c.Messages) - c.MaxMessages
	c.Messages = c.Messages[excess:]
}
