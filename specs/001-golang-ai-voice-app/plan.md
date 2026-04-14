# Implementation Plan: Golang AI Voice Conversation App

**Branch**: `001-golang-ai-voice-app` | **Date**: 2026-04-09 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-golang-ai-voice-app/spec.md`

## Summary

Build a Go web application under `voice-model/` that enables text and voice conversations with AI models. The Go backend serves a web UI on port 8080, integrates with multiple AI providers (OpenAI default, Google Gemini) via a provider abstraction layer, and handles speech-to-text (Whisper) and text-to-speech via provider APIs. Audio capture and playback are browser-side (Web Audio API / MediaRecorder). The project includes a Dockerfile (multi-stage), docker-compose.yml, and README.md, all co-located inside `voice-model/`.

## Technical Context

**Language/Version**: Go 1.22+ (latest stable with enhanced standard library router)  
**Primary Dependencies**:
- Backend HTTP: `github.com/go-chi/chi/v5` (lightweight router + middleware)
- OpenAI SDK: `github.com/sashabaranov/go-openai` (chat, Whisper STT, TTS)
- Gemini SDK: `github.com/google/generative-ai-go` (text generation only)
- WebSocket: `github.com/coder/websocket` (binary audio streaming)
- Env loading: `github.com/joho/godotenv` (.env file support)
- Frontend: Vanilla HTML/CSS/JS embedded via Go `embed.FS`, Web Audio API, MediaRecorder API
**Storage**: In-memory only (session-based conversation history; no database)  
**Testing**: `go test` + `github.com/stretchr/testify` (assertions only)  
**Target Platform**: Docker container (Linux amd64), accessed via web browser  
**Project Type**: Web application (Go backend + browser frontend)  
**Performance Goals**: <5s text response (SC-001), <8s voice round-trip (SC-002), ≥90% STT accuracy (SC-003), <3s TTS start (SC-004), 30min continuous voice session (SC-008)  
**Constraints**: Single-user v1, English only, cloud AI APIs only, port 8080, env-var secrets  
**Scale/Scope**: Single concurrent user, session-based (no persistence), ~4 pages/views (chat, settings/provider-select, error states)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Status**: PASS (N/A) — Constitution file contains only template placeholders; no project-specific principles or gates have been defined. No violations possible.

**Recommendation**: Define project-specific principles in `.specify/memory/constitution.md` before v2 development to enforce architectural constraints (e.g., simplicity, test-first, observability).

**Post-Design Re-check**: PASS (N/A) — No constitution gates defined. Design is consistent with spec requirements (19 FRs, 8 SCs). No complexity violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
voice-model/
├── main.go                    # Entry point: server setup, chi router, port 8080
├── go.mod                     # Go module (voice-model)
├── go.sum                     # Dependency checksums
├── .env.example               # Template for required environment variables
├── Dockerfile                 # Multi-stage: golang:1.22-alpine → alpine:3.19
├── docker-compose.yml         # App service, port 8080:8080, env_file: .env
├── README.md                  # Project documentation (FR-019)
│
├── internal/
│   ├── config/
│   │   └── config.go          # Load env vars (godotenv), validate API keys
│   │
│   ├── provider/
│   │   ├── provider.go        # TextProvider + VoiceProvider interfaces
│   │   ├── openai.go          # OpenAI: chat (streaming), Whisper STT, TTS
│   │   ├── gemini.go          # Gemini: chat (streaming) — text only
│   │   └── registry.go        # Provider registry, active provider selection
│   │
│   ├── conversation/
│   │   ├── conversation.go    # Conversation struct, sliding window logic
│   │   └── store.go           # In-memory conversation store (map by ID)
│   │
│   ├── handler/
│   │   ├── chat.go            # POST /api/chat, POST /api/chat/stream
│   │   ├── provider.go        # GET /api/providers, PUT /api/providers/active
│   │   ├── conversation.go    # GET /api/conversation/{id}
│   │   ├── health.go          # GET /health
│   │   └── ws.go              # GET /ws — WebSocket handler (voice + text)
│   │
│   └── retry/
│       └── retry.go           # Exponential backoff (3 attempts, FR-016)
│
├── static/                    # Embedded via //go:embed static/*
│   ├── index.html             # Single-page chat UI
│   ├── style.css              # Styles
│   └── app.js                 # WebSocket client, Web Audio API, MediaRecorder
│
└── tests/
    ├── handler_test.go        # HTTP handler unit tests
    ├── provider_test.go       # Provider interface + mock tests
    ├── conversation_test.go   # Conversation sliding window tests
    └── retry_test.go          # Retry logic tests
```

**Structure Decision**: Web application pattern with Go backend serving embedded static frontend. Code organized under `internal/` (Go convention for non-exported packages). Provider abstraction via interfaces enables multi-provider support without tight coupling. All files reside under `voice-model/` per FR-010.

## Complexity Tracking

> No constitution violations — table not applicable.
