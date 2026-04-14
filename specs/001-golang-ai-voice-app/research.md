# Research: Golang AI Voice Conversation App

**Date**: 2026-04-09 | **Branch**: `001-golang-ai-voice-app`

## R1: Go HTTP Framework

**Decision**: `github.com/go-chi/chi/v5`

**Rationale**: Chi provides first-class middleware support (logger, recoverer, CORS, timeout), clean route grouping, and named parameters while remaining lightweight (~10-15 MB stripped binary). It is `net/http` compatible, so WebSocket upgrade works seamlessly. Go 1.22+ `net/http` was considered but requires verbose manual middleware chaining; chi's built-in middleware ecosystem reduces boilerplate for logging, recovery, and CORS — all needed for a web UI serving browser requests.

**Alternatives considered**:
- Go 1.22+ `net/http` — sufficient but verbose for middleware; chosen as fallback if zero-dependency is required
- Gin — most popular but overkill; includes validation/binding/rendering not needed here
- Echo — good balance but slightly larger footprint than chi with no clear advantage for this use case

## R2: OpenAI Go SDK

**Decision**: `github.com/sashabaranov/go-openai`

**Rationale**: De facto standard Go client for OpenAI API (~13k+ GitHub stars). Covers all three required capabilities in a single library: chat completions with streaming (`CreateChatCompletionStream`), speech-to-text via Whisper (`CreateTranscription`), and text-to-speech (`CreateSpeech` returning `io.ReadCloser`). No official OpenAI Go SDK exists; this community library is production-ready and actively maintained.

**Alternatives considered**:
- Official OpenAI Go SDK — does not exist as of this date
- Direct HTTP calls — unnecessarily complex when a mature client library exists

## R3: Google Gemini Go SDK

**Decision**: `github.com/google/generative-ai-go`

**Rationale**: Official Google SDK for Gemini API. Supports text generation with streaming (`GenerateContentStream`) and multi-turn chat (`StartChat().SendMessage()`). **Critical finding**: Gemini SDK does NOT provide STT or TTS APIs. Voice I/O will always use OpenAI APIs (Whisper + TTS) regardless of which text provider is selected.

**Alternatives considered**:
- `cloud.google.com/go/ai/generativelanguage` — lower-level API client; the higher-level `generative-ai-go` is preferred
- Google Cloud Speech-to-Text + Text-to-Speech (separate services) — adds significant complexity and GCP setup; deferred to future versions

## R4: WebSocket Library

**Decision**: `github.com/coder/websocket` (formerly `nhooyr.io/websocket`)

**Rationale**: Modern, actively maintained, context-aware API that follows idiomatic Go patterns. Excellent support for binary message types needed for audio streaming. The context-based timeout handling is well-suited for long-running voice sessions (SC-008: 30-minute continuous).

**Alternatives considered**:
- `github.com/gorilla/websocket` — stable and proven but in maintenance mode (no active development). API works but lacks context support. Acceptable fallback if `coder/websocket` has issues.

## R5: Environment Variable Loading

**Decision**: `github.com/joho/godotenv`

**Rationale**: Minimal dependency that loads `.env` files for local development. In Docker, environment variables are passed via `docker-compose.yml` or `docker run -e`, making this a convenience layer. Simple API: `godotenv.Load()` then `os.Getenv()`.

**Alternatives considered**:
- Standard library `os.Getenv()` only — works for Docker but poor local dev experience without `.env` support
- `github.com/caarlos0/env` — struct-based config parsing; more complex than needed for a handful of env vars

## R6: Testing Framework

**Decision**: Standard `testing` package + `github.com/stretchr/testify` (assert/require only)

**Rationale**: Go's built-in `testing` package provides the test runner, subtests, and benchmarks. Testify adds concise assertions (`assert.Equal`, `require.NoError`) that reduce test verbosity without pulling in a heavy framework. Mocking features available but not needed initially — provider abstraction interfaces enable test doubles via plain structs.

**Alternatives considered**:
- Standard `testing` only — sufficient but verbose for complex assertions (manual `if got != want` checks)
- Full testify suite — overkill; only assertions needed for v1

## R7: Voice I/O Architecture (Multi-Provider)

**Decision**: OpenAI handles all voice I/O regardless of text provider selection.

**Rationale**: Google Gemini has no STT/TTS APIs. The provider abstraction separates concerns:
- `TextProvider` interface: implemented by both `OpenAITextProvider` and `GeminiTextProvider`
- `VoiceProvider` interface: implemented only by `OpenAIVoiceProvider` (STT via Whisper, TTS via OpenAI TTS)
- This means `OPENAI_API_KEY` is always required for voice features, even if Gemini is selected for text generation.

**Alternatives considered**:
- Google Cloud Speech-to-Text + Text-to-Speech as separate Gemini-side voice provider — adds GCP dependencies and complexity; deferred to v2

## R8: Frontend Strategy

**Decision**: Vanilla HTML/CSS/JavaScript with Go `embed.FS` for single-binary deployment.

**Rationale**: The web UI is a single-page chat interface with a voice toggle — no complex state management or routing needed. Vanilla JS with Web Audio API / MediaRecorder handles audio capture and playback. Embedding static files via `//go:embed` creates a self-contained binary ideal for Docker (no separate static file layer). This avoids Node.js build tooling entirely.

**Alternatives considered**:
- React/Vue/Svelte — adds Node.js build step, npm dependencies, and complexity for a simple chat UI
- HTMX — good for server-rendered HTML but less suitable for real-time WebSocket audio streaming
- Go templates (html/template) — suitable for initial page render but real-time updates still need JS/WebSocket

## R9: Docker Multi-Stage Build

**Decision**: Two-stage Dockerfile: Go builder (`golang:1.22-alpine`) → runtime (`alpine:3.19`).

**Rationale**: Multi-stage build produces a minimal image (~15-25 MB) containing only the compiled binary and CA certificates. Alpine base for small size. Static files embedded in binary via `embed.FS` — no file copying needed.

**Alternatives considered**:
- `scratch` base — smaller but no shell for debugging; Alpine is ~5 MB overhead but much more practical
- `distroless` — good security profile but harder to debug; Alpine sufficient for v1
- Single-stage build — results in ~1 GB image (includes full Go toolchain); unacceptable
