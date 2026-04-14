# Quickstart: Golang AI Voice Conversation App

## Prerequisites

- **Go 1.22+**: [Install Go](https://go.dev/dl/)
- **Docker & Docker Compose**: [Install Docker](https://docs.docker.com/get-docker/) (for containerized deployment)
- **OpenAI API Key**: Required for text generation, STT (Whisper), and TTS. [Get a key](https://platform.openai.com/api-keys)
- **Google Gemini API Key** (optional): Required only if using Gemini for text generation. [Get a key](https://aistudio.google.com/apikey)

## Environment Setup

Create a `.env` file inside `voice-model/`:

```bash
cd voice-model
cp .env.example .env
```

Edit `.env`:
```env
OPENAI_API_KEY=sk-your-openai-key-here
GEMINI_API_KEY=your-gemini-key-here  # optional
```

> **Important**: Never commit `.env` to version control. It is already in `.gitignore`.

## Option 1: Run Locally (Go)

```bash
cd voice-model
go mod download
go run .
```

Open `http://localhost:8080` in your browser.

## Option 2: Run with Docker Compose (Recommended)

```bash
cd voice-model
docker compose up --build
```

Open `http://localhost:8080` in your browser.

To run in the background:
```bash
docker compose up --build -d
```

To stop:
```bash
docker compose down
```

## Option 3: Build Docker Image Manually

```bash
cd voice-model
docker build -t voice-model .
docker run -p 8080:8080 --env-file .env voice-model
```

## Usage

1. **Text Chat**: Type a message in the input box and press Enter or click Send.
2. **Voice Input**: Click the microphone button to start recording. Speak your message and click again to stop. The audio is transcribed and sent to the AI.
3. **Voice Output**: When voice mode is enabled, AI responses are read aloud via text-to-speech.
4. **Toggle Voice**: Use the voice toggle button to switch between text-only and voice mode.
5. **Switch Provider**: Open settings to switch between OpenAI and Google Gemini for text generation. Note: Voice I/O always uses OpenAI regardless of text provider.

## Running Tests

```bash
cd voice-model
go test ./...
```

With verbose output:
```bash
go test -v ./...
```

## Project Structure

```
voice-model/
├── main.go              # Entry point, server setup, routing
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── .env.example         # Template for environment variables
├── Dockerfile           # Multi-stage Docker build
├── docker-compose.yml   # Docker Compose service definition
├── README.md            # Project documentation
├── internal/
│   ├── config/          # Configuration loading (env vars)
│   ├── provider/        # AI provider abstraction layer
│   │   ├── provider.go  # TextProvider and VoiceProvider interfaces
│   │   ├── openai.go    # OpenAI implementation (text + voice)
│   │   └── gemini.go    # Gemini implementation (text only)
│   ├── conversation/    # Conversation and message management
│   ├── handler/         # HTTP and WebSocket handlers
│   └── retry/           # Exponential backoff retry logic
├── static/
│   ├── index.html       # Main chat UI
│   ├── style.css        # Styles
│   └── app.js           # Frontend logic (WebSocket, Web Audio API)
└── tests/
    ├── handler_test.go  # HTTP handler tests
    ├── provider_test.go # Provider abstraction tests
    └── conversation_test.go # Conversation logic tests
```
