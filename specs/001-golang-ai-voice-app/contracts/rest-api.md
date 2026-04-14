# API Contracts: REST Endpoints

**Date**: 2026-04-09 | **Branch**: `001-golang-ai-voice-app`

All endpoints served on `http://localhost:8080`. JSON request/response unless noted.

---

## POST /api/chat

Send a text message to the AI and receive a response.

**Request**:
```json
{
  "message": "string (required, non-empty)",
  "conversation_id": "string (optional, UUID — omit to create new session)"
}
```

**Response** (200 OK):
```json
{
  "conversation_id": "string (UUID)",
  "message_id": "string (UUID)",
  "content": "string (AI response text)",
  "input_mode": "text",
  "timestamp": "string (RFC3339)"
}
```

**Error Responses**:
- `400 Bad Request` — empty message or invalid JSON
  ```json
  { "error": "message is required" }
  ```
- `502 Bad Gateway` — AI provider unreachable after retries (FR-016)
  ```json
  { "error": "AI service unavailable, please retry", "retryable": true }
  ```
- `429 Too Many Requests` — rate limited after retries exhausted
  ```json
  { "error": "rate limit exceeded, please wait", "retryable": true }
  ```

---

## POST /api/chat/stream

Send a text message and receive a streamed response via Server-Sent Events (SSE).

**Request**: Same as `POST /api/chat`.

**Response** (200 OK, `Content-Type: text/event-stream`):
```
data: {"type": "chunk", "content": "partial text..."}

data: {"type": "chunk", "content": "more text..."}

data: {"type": "done", "message_id": "uuid", "conversation_id": "uuid"}

```

**Error**: SSE event with error type:
```
data: {"type": "error", "error": "AI service unavailable"}

```

---

## GET /api/providers

List available AI text providers and the currently active one.

**Response** (200 OK):
```json
{
  "active": "openai",
  "providers": [
    { "name": "openai", "available": true },
    { "name": "gemini", "available": false }
  ]
}
```

`available` is `true` if the provider's API key is configured.

---

## PUT /api/providers/active

Switch the active AI text provider (FR-014).

**Request**:
```json
{
  "provider": "string (required: 'openai' | 'gemini')"
}
```

**Response** (200 OK):
```json
{
  "active": "gemini",
  "message": "Provider switched to gemini"
}
```

**Error**:
- `400 Bad Request` — unknown provider name
- `422 Unprocessable Entity` — provider not available (API key missing)
  ```json
  { "error": "gemini provider not configured (GEMINI_API_KEY not set)" }
  ```

---

## GET /api/conversation/{id}

Retrieve conversation history for display (FR-003, User Story 4).

**Response** (200 OK):
```json
{
  "conversation_id": "string (UUID)",
  "provider": "openai",
  "messages": [
    {
      "id": "string (UUID)",
      "role": "user",
      "content": "Hello",
      "input_mode": "text",
      "timestamp": "2026-04-09T10:30:00Z"
    },
    {
      "id": "string (UUID)",
      "role": "assistant",
      "content": "Hi! How can I help?",
      "input_mode": "text",
      "timestamp": "2026-04-09T10:30:02Z"
    }
  ]
}
```

**Error**:
- `404 Not Found` — conversation ID not found

---

## GET /health

Health check endpoint for Docker and monitoring.

**Response** (200 OK):
```json
{
  "status": "ok",
  "provider": "openai",
  "version": "1.0.0"
}
```

---

## Static Files

| Route | Description |
|-------|-------------|
| `GET /` | Serves `index.html` (main chat UI) |
| `GET /static/*` | Serves embedded static assets (CSS, JS) |
