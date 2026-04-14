package provider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
	model  string
}

func NewOpenAI(apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) SendMessage(ctx context.Context, messages []Message, onChunk func(string)) (string, error) {
	msgs := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	if onChunk != nil {
		return p.sendStreaming(ctx, msgs, onChunk)
	}
	return p.sendNonStreaming(ctx, msgs)
}

func (p *OpenAIProvider) sendNonStreaming(ctx context.Context, msgs []openai.ChatCompletionMessage) (string, error) {
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    p.model,
		Messages: msgs,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("no response from OpenAI")
	}
	return resp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) sendStreaming(ctx context.Context, msgs []openai.ChatCompletionMessage, onChunk func(string)) (string, error) {
	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    p.model,
		Messages: msgs,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var full strings.Builder
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return full.String(), err
		}
		chunk := resp.Choices[0].Delta.Content
		if chunk != "" {
			full.WriteString(chunk)
			onChunk(chunk)
		}
	}
	return full.String(), nil
}

// Transcribe implements VoiceProvider — speech-to-text via Whisper.
func (p *OpenAIProvider) Transcribe(ctx context.Context, audio io.Reader, format string) (string, error) {
	buf, err := io.ReadAll(audio)
	if err != nil {
		return "", err
	}

	ext := format
	if ext == "webm" {
		ext = "webm"
	}

	resp, err := p.client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   bytes.NewReader(buf),
		FilePath: "audio." + ext,
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// Synthesize implements VoiceProvider — text-to-speech via OpenAI TTS.
func (p *OpenAIProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, error) {
	resp, err := p.client.CreateSpeech(ctx, openai.CreateSpeechRequest{
		Model:          openai.TTSModel1,
		Voice:          openai.VoiceAlloy,
		Input:          text,
		ResponseFormat: openai.SpeechResponseFormatMp3,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
