# Data Model: Golang AI Voice Conversation App

**Date**: 2026-04-09 | **Branch**: `001-golang-ai-voice-app`

## Entities

### Conversation

Represents a single chat session between the user and the AI. Exists only in memory; discarded when the session ends or the server restarts.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique session identifier (UUID v4). |
| `Messages` | `[]Message` | Ordered list of messages in the conversation. |
| `Provider` | `string` | Active AI text provider name (`"openai"` or `"gemini"`). |
| `CreatedAt` | `time.Time` | Timestamp when the session was created. |
| `SystemPrompt` | `string` | System prompt always preserved during sliding window truncation. |
| `MaxMessages` | `int` | Configurable sliding window size (default: 50). |

**Identity / Uniqueness**: `ID` is globally unique (UUID v4 generated at session creation).

**Lifecycle**:
```
Created → Active → Ended
```
- **Created**: On first HTTP request or WebSocket connection from a new session.
- **Active**: Messages are appended; sliding window applied when `len(Messages) > MaxMessages`.
- **Ended**: Server restart, explicit session end, or inactivity timeout (if implemented).

---

### Message

A single unit of communication within a Conversation.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique message identifier (UUID v4). |
| `Role` | `string` | Sender role: `"user"`, `"assistant"`, or `"system"`. |
| `Content` | `string` | Text content of the message. |
| `InputMode` | `string` | How the message was created: `"text"` or `"voice"`. |
| `Timestamp` | `time.Time` | When the message was created. |

**Validation rules**:
- `Content` must not be empty for `"user"` role (FR-001, acceptance scenario 3).
- `Role` must be one of `"user"`, `"assistant"`, `"system"`.
- `InputMode` must be `"text"` or `"voice"`.

---

### ProviderConfig

Configuration for an AI text generation provider.

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Provider identifier: `"openai"` or `"gemini"`. |
| `APIKey` | `string` | API key loaded from environment variable. Never persisted. |
| `Model` | `string` | Model name (e.g., `"gpt-4"`, `"gemini-pro"`). |
| `Available` | `bool` | Whether the provider's API key is configured and non-empty. |

**Validation rules**:
- `APIKey` must be non-empty for the provider to be `Available`.
- `Model` defaults to a sensible default per provider if not explicitly set.

---

### VoiceConfig

Configuration for voice I/O services (always OpenAI in v1).

| Field | Type | Description |
|-------|------|-------------|
| `STTModel` | `string` | Speech-to-text model (default: `"whisper-1"`). |
| `TTSModel` | `string` | Text-to-speech model (default: `"tts-1"`). |
| `TTSVoice` | `string` | TTS voice name (default: `"alloy"`). |
| `AudioFormat` | `string` | Audio format for TTS output (default: `"mp3"`). |

---

## Relationships

```
Conversation 1──* Message
    │
    └── has ProviderConfig (active provider)

VoiceConfig (singleton, app-level)
    └── always uses OpenAI API key from ProviderConfig["openai"]
```

- One `Conversation` contains zero or more `Message` entities (ordered by `Timestamp`).
- Each `Conversation` references a `ProviderConfig` by `Provider` name.
- `VoiceConfig` is application-level (not per-conversation); uses the OpenAI API key regardless of which text provider is active.

## Interfaces (Provider Abstraction)

### TextProvider

```go
type TextProvider interface {
    // SendMessage sends a user message with conversation history and returns the assistant's response.
    // Supports streaming via the callback; if callback is nil, returns the full response.
    SendMessage(ctx context.Context, messages []Message, onChunk func(chunk string)) (string, error)
    
    // Name returns the provider identifier ("openai" or "gemini").
    Name() string
}
```

**Implementations**: `OpenAITextProvider`, `GeminiTextProvider`

### VoiceProvider

```go
type VoiceProvider interface {
    // Transcribe converts audio data to text (STT).
    Transcribe(ctx context.Context, audio io.Reader, format string) (string, error)
    
    // Synthesize converts text to audio data (TTS), returning an audio stream.
    Synthesize(ctx context.Context, text string) (io.ReadCloser, error)
}
```

**Implementations**: `OpenAIVoiceProvider` (only implementation in v1)

## State Transitions

### Conversation Lifecycle

```
[New Session] ──► Created
       │
       ▼
     Active ◄────────────────┐
       │                     │
       ├── user sends msg ──►│ (append Message, maybe slide window)
       │                     │
       ├── provider switch ──│ (update Provider field)
       │                     │
       └── end/timeout ─────► Ended (discard from memory)
```

### Voice Input Flow

```
[Browser: Recording] ──► [Browser: Send audio via WebSocket]
       │
       ▼
[Server: Receive audio] ──► [Server: Call Whisper STT]
       │
       ▼
[Server: Get transcription] ──► [Server: Send to TextProvider]
       │
       ▼
[Server: Get AI response] ──► [Server: Call TTS] ──► [Server: Stream audio via WebSocket]
       │                                                      │
       ▼                                                      ▼
[Server: Send text via WebSocket]              [Browser: Play audio]
```
