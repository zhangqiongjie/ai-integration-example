# AI Voice Conversation App

A Go web application that enables text and voice conversations with AI models (OpenAI, Google Gemini) through a browser-based interface.

## Features

- **Multi-turn text chat** with AI via OpenAI GPT or Google Gemini
- **Voice input** — speak into your microphone, transcribed via OpenAI Whisper
- **Voice output** — AI responses spoken aloud via OpenAI TTS
- **Provider switching** — switch between OpenAI and Gemini at runtime
- **Streaming responses** — AI text streams in real-time via WebSocket
- **Conversation history** — maintained in-memory for the session
- **Sliding window** — automatically trims old messages, preserving system prompt
- **Single binary** — frontend embedded via Go `embed.FS`

## Prerequisites

- **Docker** and **Docker Compose** (recommended), OR **Go 1.22+** for local development
- **OpenAI API key** (required for all features)
- **Google Gemini API key** (optional, for text-only alternative provider)

## Quick Start (Docker)

```bash
# 1. Copy environment template and add your API keys
cp .env.example .env
# Edit .env and set OPENAI_API_KEY (required) and GEMINI_API_KEY (optional)

# 2. Build and run
docker compose up --build

# 3. Open http://localhost:8080 in your browser
```

## Quick Start (Local Go)

```bash
# 1. Copy environment template and add your API keys
cp .env.example .env

# 2. Install dependencies
go mod download

# 3. Run the server
go run .

# 4. Open http://localhost:8080 in your browser
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | Yes | — | OpenAI API key for chat, STT, and TTS |
| `GEMINI_API_KEY` | No | — | Google Gemini API key (text-only) |
| `PORT` | No | `8080` | HTTP server port |
| `OPENAI_MODEL` | No | `gpt-4` | OpenAI model for text generation |
| `GEMINI_MODEL` | No | `gemini-pro` | Gemini model for text generation |
| `DEFAULT_PROVIDER` | No | `openai` | Default text provider |
| `MAX_MESSAGES` | No | `50` | Sliding window size |
| `SYSTEM_PROMPT` | No | `You are a helpful AI assistant.` | System prompt |

## Usage

### Text Chat
1. Type a message in the input field and press Enter or click Send.
2. The AI response streams in real-time.

### Voice Input
1. Hold the microphone button to record.
2. Release to send audio to Whisper for transcription.
3. Auto-stops after 1.5 seconds of silence.
4. Falls back to text input if microphone is denied.

### Voice Output
1. Click the speaker icon to toggle voice output on/off.
2. When enabled, AI responses are spoken aloud via TTS.

### Provider Switching
1. Use the dropdown in the header to switch between OpenAI and Gemini.
2. Voice I/O always uses OpenAI regardless of text provider selection.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/chat` | Send message, get full response |
| `POST` | `/api/chat/stream` | Send message, get SSE stream |
| `GET` | `/api/providers` | List providers |
| `PUT` | `/api/providers/active` | Switch provider |
| `GET` | `/api/conversation/{id}` | Get conversation history |
| `GET` | `/ws` | WebSocket endpoint |

## Project Structure

```
voice-model/
├── main.go                     # Entry point, chi router, embedded static files
├── go.mod / go.sum             # Go module
├── .env.example                # Environment variable template
├── Dockerfile                  # Multi-stage Docker build
├── docker-compose.yml          # Docker Compose config
├── internal/
│   ├── config/config.go        # Environment loading, ProviderConfig, VoiceConfig
│   ├── provider/
│   │   ├── provider.go         # TextProvider + VoiceProvider interfaces
│   │   ├── openai.go           # OpenAI: chat, Whisper STT, TTS
│   │   ├── gemini.go           # Gemini: chat (text only)
│   │   └── registry.go         # Provider registry with retry
│   ├── conversation/
│   │   ├── conversation.go     # Conversation + Message structs, sliding window
│   │   └── store.go            # Thread-safe in-memory store
│   ├── handler/
│   │   ├── health.go           # GET /health
│   │   ├── chat.go             # POST /api/chat, POST /api/chat/stream
│   │   ├── provider.go         # GET /api/providers, PUT /api/providers/active
│   │   ├── conversation.go     # GET /api/conversation/{id}
│   │   └── ws.go               # WebSocket handler (text + voice)
│   └── retry/retry.go          # Exponential backoff (3 attempts)
└── static/
    ├── index.html              # Chat UI
    ├── style.css               # Dark theme styles
    └── app.js                  # WebSocket client, voice capture, audio playback
```

## Docker

```bash
# Build image
docker build -t voice-model .

# Run container
docker run -p 8080:8080 --env-file .env voice-model

# Or use Docker Compose
docker compose up --build
```

## Architecture

- **Backend**: Go 1.22+ with chi router, WebSocket via `coder/websocket`
- **Frontend**: Vanilla HTML/CSS/JS embedded in the Go binary
- **AI Providers**: Abstracted via `TextProvider`/`VoiceProvider` interfaces
- **Voice**: Browser captures audio via MediaRecorder API; server handles STT/TTS via OpenAI
- **Storage**: In-memory only (session-based, no database)
- **Retry**: Exponential backoff (3 attempts) on AI provider errors
