# API Contracts: WebSocket Protocol

**Date**: 2026-04-09 | **Branch**: `001-golang-ai-voice-app`

## Endpoint

```
ws://localhost:8080/ws?conversation_id={uuid}
```

- `conversation_id` (optional): attach to existing conversation. Omit to create a new one.
- On successful connection, the server sends an `init` message with the conversation ID.

---

## Message Format

All text-frame messages are JSON. Binary frames carry raw audio data.

### Client → Server Messages

#### Text Message (JSON text frame)

```json
{
  "type": "text",
  "content": "Hello, how are you?"
}
```

#### Voice Start (JSON text frame)

Signals the client is about to send audio data.

```json
{
  "type": "voice_start",
  "format": "webm"
}
```

- `format`: audio codec of the incoming binary frames (`"webm"`, `"ogg"`, `"wav"`).

#### Voice Audio (binary frame)

Raw audio data chunks from the browser's MediaRecorder. Sent as binary WebSocket frames during recording.

#### Voice End (JSON text frame)

Signals the client has finished recording.

```json
{
  "type": "voice_end"
}
```

#### Provider Switch (JSON text frame)

```json
{
  "type": "provider_switch",
  "provider": "gemini"
}
```

#### Voice Toggle (JSON text frame)

```json
{
  "type": "voice_toggle",
  "enabled": true
}
```

Toggles whether AI responses should include TTS audio (FR-006).

---

### Server → Client Messages

#### Init (JSON text frame)

Sent immediately on connection.

```json
{
  "type": "init",
  "conversation_id": "uuid",
  "provider": "openai",
  "voice_enabled": true
}
```

#### Transcription (JSON text frame)

Speech-to-text result after voice input is processed.

```json
{
  "type": "transcription",
  "content": "What is the weather today?",
  "message_id": "uuid"
}
```

#### Response Chunk (JSON text frame)

Streamed AI text response (one chunk at a time).

```json
{
  "type": "response_chunk",
  "content": "partial response text..."
}
```

#### Response Done (JSON text frame)

Signals the complete AI response has been sent.

```json
{
  "type": "response_done",
  "message_id": "uuid",
  "full_content": "The complete AI response text."
}
```

#### TTS Audio (binary frame)

Raw audio data for text-to-speech playback. Sent as binary WebSocket frames after `response_done` when voice output is enabled.

#### TTS Done (JSON text frame)

Signals all TTS audio has been sent.

```json
{
  "type": "tts_done"
}
```

#### Error (JSON text frame)

```json
{
  "type": "error",
  "error": "AI service unavailable",
  "retryable": true
}
```

#### Provider Changed (JSON text frame)

Confirmation of provider switch.

```json
{
  "type": "provider_changed",
  "provider": "gemini"
}
```

---

## Connection Lifecycle

```
Client                          Server
  │                               │
  │──── WS Connect ──────────────►│
  │◄─── init ────────────────────│
  │                               │
  │──── text / voice_start ──────►│
  │──── [binary audio frames] ──►│ (only if voice)
  │──── voice_end ───────────────►│ (only if voice)
  │                               │
  │◄─── transcription ───────────│ (only if voice)
  │◄─── response_chunk ──────────│ (streamed, multiple)
  │◄─── response_done ───────────│
  │◄─── [binary TTS frames] ─────│ (only if voice enabled)
  │◄─── tts_done ─────────────── │ (only if voice enabled)
  │                               │
  │──── WS Close ────────────────►│
  │◄─── WS Close ────────────────│
```

## Error Handling

- If the AI provider returns an error, the server sends an `error` message and keeps the connection open.
- If the WebSocket connection drops, the browser should attempt reconnection with the same `conversation_id`.
- Rate-limit errors from the AI provider trigger automatic retry (up to 3 attempts with exponential backoff) before an `error` message is sent to the client.
- Simultaneous voice input during TTS playback: the server cancels any in-progress TTS streaming when a new `voice_start` is received (edge case per spec).
