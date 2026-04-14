package provider

import (
	"context"
	"errors"
	"io"
	"strings"

	"google.golang.org/api/iterator"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	apiKey string
	model  string
}

func NewGemini(apiKey, model string) *GeminiProvider {
	return &GeminiProvider{
		apiKey: apiKey,
		model:  model,
	}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) SendMessage(ctx context.Context, messages []Message, onChunk func(string)) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(p.apiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(p.model)
	cs := model.StartChat()

	// Build history from prior messages (skip system and last user message).
	var lastUserContent string
	for _, m := range messages {
		switch m.Role {
		case "system":
			model.SystemInstruction = genai.NewUserContent(genai.Text(m.Content))
		case "user":
			lastUserContent = m.Content
			cs.History = append(cs.History, &genai.Content{
				Parts: []genai.Part{genai.Text(m.Content)},
				Role:  "user",
			})
		case "assistant":
			cs.History = append(cs.History, &genai.Content{
				Parts: []genai.Part{genai.Text(m.Content)},
				Role:  "model",
			})
		}
	}

	// Remove the last user message from history since we'll send it as the prompt.
	if len(cs.History) > 0 && cs.History[len(cs.History)-1].Role == "user" {
		cs.History = cs.History[:len(cs.History)-1]
	}

	if lastUserContent == "" {
		return "", errors.New("no user message found")
	}

	if onChunk != nil {
		return p.sendStreaming(ctx, cs, lastUserContent, onChunk)
	}
	return p.sendNonStreaming(ctx, cs, lastUserContent)
}

func (p *GeminiProvider) sendNonStreaming(ctx context.Context, cs *genai.ChatSession, prompt string) (string, error) {
	resp, err := cs.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}
	return extractText(resp), nil
}

func (p *GeminiProvider) sendStreaming(ctx context.Context, cs *genai.ChatSession, prompt string, onChunk func(string)) (string, error) {
	iter := cs.SendMessageStream(ctx, genai.Text(prompt))
	var full strings.Builder
	for {
		resp, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return full.String(), err
		}
		chunk := extractText(resp)
		if chunk != "" {
			full.WriteString(chunk)
			onChunk(chunk)
		}
	}
	return full.String(), nil
}

func extractText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return ""
	}
	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			sb.WriteString(string(t))
		}
	}
	return sb.String()
}

// Ensure GeminiProvider does NOT implement VoiceProvider (text-only).
var _ TextProvider = (*GeminiProvider)(nil)

// Verify OpenAI implements both interfaces (compile-time check, placed here for convenience).
var _ TextProvider = (*OpenAIProvider)(nil)
var _ VoiceProvider = (*OpenAIProvider)(nil)

// Unused import guard.
var _ = io.EOF
