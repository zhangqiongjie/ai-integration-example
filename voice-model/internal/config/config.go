package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ProviderConfig struct {
	Name      string
	APIKey    string
	Model     string
	Available bool
}

type VoiceConfig struct {
	STTModel    string
	TTSModel    string
	TTSVoice    string
	AudioFormat string
}

type Config struct {
	Port          string
	OpenAI        ProviderConfig
	Gemini        ProviderConfig
	Voice         VoiceConfig
	MaxMessages   int
	SystemPrompt  string
	DefaultProvider string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Port: getEnv("PORT", "8080"),
		OpenAI: ProviderConfig{
			Name:   "openai",
			APIKey: os.Getenv("OPENAI_API_KEY"),
			Model:  getEnv("OPENAI_MODEL", "gpt-4"),
		},
		Gemini: ProviderConfig{
			Name:   "gemini",
			APIKey: os.Getenv("GEMINI_API_KEY"),
			Model:  getEnv("GEMINI_MODEL", "gemini-pro"),
		},
		Voice: VoiceConfig{
			STTModel:    getEnv("STT_MODEL", "whisper-1"),
			TTSModel:    getEnv("TTS_MODEL", "tts-1"),
			TTSVoice:    getEnv("TTS_VOICE", "alloy"),
			AudioFormat: getEnv("AUDIO_FORMAT", "mp3"),
		},
		MaxMessages:   getEnvInt("MAX_MESSAGES", 50),
		SystemPrompt:  getEnv("SYSTEM_PROMPT", "You are a helpful AI assistant."),
		DefaultProvider: getEnv("DEFAULT_PROVIDER", "openai"),
	}

	cfg.OpenAI.Available = cfg.OpenAI.APIKey != ""
	cfg.Gemini.Available = cfg.Gemini.APIKey != ""

	if !cfg.OpenAI.Available {
		log.Println("WARNING: OPENAI_API_KEY not set — OpenAI provider unavailable")
	}
	if !cfg.Gemini.Available {
		log.Println("INFO: GEMINI_API_KEY not set — Gemini provider unavailable")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
