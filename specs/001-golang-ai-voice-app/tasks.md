# Tasks: Golang AI Voice Conversation App

**Input**: Design documents from `/specs/001-golang-ai-voice-app/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in the feature specification. Test tasks are omitted. Test files are listed in plan.md project structure for future addition.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

All paths are relative to the repository root. The application resides under `voice-model/` per FR-010.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, directory structure, Docker configuration

- [x] T001 Create voice-model/ directory structure with subdirectories: internal/config, internal/provider, internal/conversation, internal/handler, internal/retry, static, tests
- [x] T002 Initialize Go module and install all dependencies (chi, go-openai, generative-ai-go, coder/websocket, godotenv, testify) in voice-model/go.mod
- [x] T003 [P] Create environment variable template with OPENAI_API_KEY and GEMINI_API_KEY in voice-model/.env.example
- [x] T004 [P] Create multi-stage Dockerfile (golang:1.22-alpine builder → alpine:3.19 runtime, EXPOSE 8080) in voice-model/Dockerfile
- [x] T005 [P] Create docker-compose.yml with app service, port 8080:8080, and env_file .env in voice-model/docker-compose.yml

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 Implement configuration loading from environment variables with godotenv .env fallback, validate API keys, and expose ProviderConfig/VoiceConfig structs in voice-model/internal/config/config.go
- [x] T007 [P] Define TextProvider interface (SendMessage with streaming callback, Name) and VoiceProvider interface (Transcribe, Synthesize) in voice-model/internal/provider/provider.go
- [x] T008 [P] Implement exponential backoff retry utility (max 3 attempts, configurable base delay, context-aware cancellation) in voice-model/internal/retry/retry.go
- [x] T009 [P] Implement Conversation struct (ID, Messages, Provider, SystemPrompt, MaxMessages) and Message struct with sliding window logic (preserve system prompt, drop oldest) in voice-model/internal/conversation/conversation.go
- [x] T010 [P] Implement thread-safe in-memory conversation store (Create, Get, Delete by ID, sync.RWMutex) in voice-model/internal/conversation/store.go
- [x] T011 Implement chi router with middleware (logger, recoverer, CORS), embed.FS static file serving, and route registration for all API endpoints in voice-model/main.go
- [x] T012 [P] Implement GET /health endpoint returning status, active provider, and version in voice-model/internal/handler/health.go

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Text-Based AI Conversation (Priority: P1) MVP

**Goal**: Users can send text messages via a web UI and receive AI-generated responses with multi-turn context, using either OpenAI or Gemini as the text provider.

**Independent Test**: Launch the app, open http://localhost:8080, type a message, verify a coherent AI response appears within 5 seconds. Send a follow-up message and verify the AI references prior context. Send an empty message and verify a validation prompt appears.

**FRs**: FR-001, FR-002, FR-003, FR-007, FR-012, FR-013, FR-014, FR-015, FR-016
**Contracts**: POST /api/chat, POST /api/chat/stream, GET /api/providers, PUT /api/providers/active, WS text/response_chunk/response_done/provider_switch

### Implementation for User Story 1

- [x] T013 [P] [US1] Implement OpenAI text provider with CreateChatCompletionStream, message history conversion, and streaming callback in voice-model/internal/provider/openai.go
- [x] T014 [P] [US1] Implement Gemini text provider with GenerateContentStream, chat session management, and streaming callback in voice-model/internal/provider/gemini.go
- [x] T015 [US1] Implement provider registry (register providers, get/set active, list available) with retry wrapper around SendMessage in voice-model/internal/provider/registry.go
- [x] T016 [US1] Implement POST /api/chat handler: parse request, validate non-empty message, call TextProvider.SendMessage, return JSON response per rest-api.md contract in voice-model/internal/handler/chat.go
- [x] T017 [US1] Implement POST /api/chat/stream handler: SSE response with chunk/done/error events per rest-api.md contract in voice-model/internal/handler/chat.go
- [x] T018 [P] [US1] Implement GET /api/providers (list with availability) and PUT /api/providers/active (switch with validation) per rest-api.md contract in voice-model/internal/handler/provider.go
- [x] T019 [US1] Implement WebSocket handler: accept connection, send init message, handle text messages, stream response_chunk/response_done, handle provider_switch per websocket-protocol.md in voice-model/internal/handler/ws.go
- [x] T020 [P] [US1] Create chat UI HTML with message list, text input with send button, provider selector dropdown, and connection status indicator in voice-model/static/index.html
- [x] T021 [P] [US1] Create chat UI styles: conversation layout, user/assistant message bubbles, input area, provider selector, responsive design in voice-model/static/style.css
- [x] T022 [US1] Implement frontend WebSocket client: connect to /ws, send text messages, render streamed response chunks, handle provider switching, display errors, and empty message validation in voice-model/static/app.js

**Checkpoint**: User Story 1 fully functional — text chat with multi-turn context, provider switching, streaming responses, error handling. Independently testable.

---

## Phase 4: User Story 2 — Voice Input to AI (Priority: P2)

**Goal**: Users can speak into their microphone and have their speech transcribed by Whisper, sent to the AI, and receive a text response — enabling hands-free input.

**Independent Test**: Click the microphone button, speak a question, verify the transcription appears in the chat, and confirm the AI generates a relevant text response based on the transcribed text. Test with mic denied to verify fallback to text input with error message.

**FRs**: FR-004, FR-008, FR-009
**Contracts**: WS voice_start, binary audio frames, voice_end, transcription

### Implementation for User Story 2

- [x] T023 [US2] Implement OpenAI Whisper STT (Transcribe method): accept io.Reader audio + format, call CreateTranscription, return transcribed text in voice-model/internal/provider/openai.go
- [x] T024 [US2] Add WebSocket voice input handling: accumulate binary audio frames between voice_start and voice_end, call VoiceProvider.Transcribe, send transcription message, then forward to TextProvider in voice-model/internal/handler/ws.go
- [x] T025 [US2] Add browser microphone capture: request getUserMedia permission, create MediaRecorder (webm format), send voice_start on record, stream binary chunks via WebSocket, send voice_end on stop in voice-model/static/app.js
- [x] T026 [US2] Implement end-of-speech detection: monitor audio levels via Web Audio API AnalyserNode, trigger auto-stop after configurable silence threshold (e.g., 1.5s) in voice-model/static/app.js
- [x] T027 [US2] Add voice input error handling: detect mic permission denied and display fallback-to-text message, handle STT errors (unintelligible speech) with retry/text-fallback prompt in voice-model/internal/handler/ws.go and voice-model/static/app.js

**Checkpoint**: User Stories 1 AND 2 both work independently — text and voice input available

---

## Phase 5: User Story 3 — Voice Output from AI (Priority: P3)

**Goal**: AI responses are spoken aloud via text-to-speech, completing the fully hands-free voice conversation loop. Users can toggle voice output on/off.

**Independent Test**: Send a text message with voice mode enabled, verify the AI response is displayed as text AND played as audible speech. Toggle voice mode off and verify only text is displayed. Test with audio device unavailable to verify text-only fallback.

**FRs**: FR-005, FR-006
**Contracts**: WS voice_toggle, binary TTS frames, tts_done

### Implementation for User Story 3

- [x] T028 [US3] Implement OpenAI TTS (Synthesize method): accept text, call CreateSpeech with tts-1 model and alloy voice, return io.ReadCloser audio stream in voice-model/internal/provider/openai.go
- [x] T029 [US3] Add WebSocket TTS streaming: after response_done, if voice enabled, call VoiceProvider.Synthesize and stream binary audio frames followed by tts_done message; handle voice_toggle messages in voice-model/internal/handler/ws.go
- [x] T030 [US3] Add browser audio playback: receive binary WebSocket frames, buffer into audio blob, play via HTML5 Audio API, show playback indicator, handle voice toggle UI button in voice-model/static/app.js
- [x] T031 [US3] Handle simultaneous voice input during TTS playback: cancel in-progress TTS streaming when new voice_start is received, stop browser audio playback on mic activation in voice-model/internal/handler/ws.go and voice-model/static/app.js

**Checkpoint**: Full voice conversation loop working — speak to AI, hear response, toggle modes

---

## Phase 6: User Story 4 — Conversation History Management (Priority: P4)

**Goal**: Conversation messages are preserved during a session for review, displayed in chronological order, and start fresh on new sessions.

**Independent Test**: Have a multi-turn conversation, scroll up to verify all prior messages are visible in order. Close the browser tab, reopen http://localhost:8080, and verify a fresh empty conversation starts.

**FRs**: FR-003 (display aspect)
**Contracts**: GET /api/conversation/{id}

### Implementation for User Story 4

- [x] T032 [US4] Implement GET /api/conversation/{id} handler: return full message history with metadata per rest-api.md contract in voice-model/internal/handler/conversation.go
- [x] T033 [US4] Add conversation history display: render all messages in chronological order with role indicators and timestamps, auto-scroll to latest message on new response in voice-model/static/app.js
- [x] T034 [US4] Implement fresh session on new browser tab: generate new conversation_id on page load (no localStorage persistence), connect WebSocket with new session in voice-model/static/app.js

**Checkpoint**: All user stories independently functional

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, edge case coverage, Docker validation

- [x] T035 [P] Create comprehensive README.md covering: project overview, prerequisites, environment setup, local dev instructions, Docker usage (build/run/compose), provider configuration, usage guide, and project structure per FR-019 in voice-model/README.md
- [x] T036 Review and harden error handling for all edge cases: network disconnection (WS reconnect with same conversation_id), overly long messages (client-side length validation), context window overflow notification to user, rate-limit user feedback across voice-model/internal/handler/
- [ ] T037 Validate Docker build and docker-compose startup: build image, verify size < 50MB, run container, confirm http://localhost:8080 serves UI, verify health endpoint responds in voice-model/
- [ ] T038 Run quickstart.md end-to-end validation: follow setup steps, test local Go run, test Docker Compose up, verify text chat works in voice-model/

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational — delivers MVP
- **User Story 2 (Phase 4)**: Depends on Foundational + US1 WebSocket handler (T019)
- **User Story 3 (Phase 5)**: Depends on Foundational + US1 WebSocket handler (T019)
- **User Story 4 (Phase 6)**: Depends on Foundational + US1 frontend (T022)
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

```
Phase 1 (Setup)
    │
    ▼
Phase 2 (Foundational) ──── BLOCKS ALL STORIES
    │
    ▼
Phase 3 (US1: Text Chat) ← MVP STOP POINT
    │
    ├──► Phase 4 (US2: Voice Input) ──► can start after T019
    │
    ├──► Phase 5 (US3: Voice Output) ──► can start after T019
    │
    └──► Phase 6 (US4: History) ──► can start after T022
              │
              ▼
         Phase 7 (Polish)
```

### Within Each User Story

- Provider implementations before registry (T013/T014 → T015)
- Backend handlers before frontend (T016-T019 → T020-T022)
- Core implementation before error handling (T023-T025 → T026-T027)

### Parallel Opportunities

**Phase 1** (3 parallel):
```
T003 (.env.example) ║ T004 (Dockerfile) ║ T005 (docker-compose.yml)
```

**Phase 2** (5 parallel):
```
T007 (interfaces) ║ T008 (retry) ║ T009 (conversation) ║ T010 (store) ║ T012 (health)
```

**Phase 3 — US1** (parallel groups):
```
Group A: T013 (OpenAI provider) ║ T014 (Gemini provider) ║ T018 (provider handler) ║ T020 (HTML) ║ T021 (CSS)
Group B: T015 (registry) → T016 (chat handler) → T017 (stream handler) → T019 (WS handler) → T022 (app.js)
```

**Phase 4-6**: Mostly sequential within each story, but US2/US3/US4 phases can run in parallel if multiple developers are available.

---

## Parallel Example: User Story 1

```bash
# Launch all parallelizable US1 tasks together:
Task: T013 "Implement OpenAI text provider in voice-model/internal/provider/openai.go"
Task: T014 "Implement Gemini text provider in voice-model/internal/provider/gemini.go"
Task: T018 "Implement provider handlers in voice-model/internal/handler/provider.go"
Task: T020 "Create chat UI HTML in voice-model/static/index.html"
Task: T021 "Create chat UI styles in voice-model/static/style.css"

# Then sequential chain:
Task: T015 → T016 → T017 → T019 → T022
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T012)
3. Complete Phase 3: User Story 1 (T013-T022)
4. **STOP and VALIDATE**: Test text chat independently at http://localhost:8080
5. Deploy/demo if ready — working AI chat app with provider switching

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → **MVP!** (text chat)
3. Add User Story 2 → Test independently → Voice input enabled
4. Add User Story 3 → Test independently → Full voice conversation
5. Add User Story 4 → Test independently → History management
6. Polish → README, Docker validation, edge cases

### Parallel Team Strategy

With multiple developers after Foundational phase:
- Developer A: User Story 1 (MVP — highest priority)
- After US1 WebSocket handler (T019) is done:
  - Developer B: User Story 2 (Voice Input)
  - Developer C: User Story 3 (Voice Output)
  - Developer A: User Story 4 (History)

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Voice I/O always uses OpenAI API regardless of text provider selection (Gemini has no STT/TTS)
- OPENAI_API_KEY is required for all features; GEMINI_API_KEY is optional (text-only alternative)
