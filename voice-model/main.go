package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"voice-model/internal/config"
	"voice-model/internal/conversation"
	"voice-model/internal/handler"
	"voice-model/internal/provider"
	"voice-model/internal/retry"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	cfg := config.Load()

	store := conversation.NewStore()
	reg := provider.NewRegistry(retry.DefaultConfig())

	if cfg.OpenAI.Available {
		oai := provider.NewOpenAI(cfg.OpenAI.APIKey, cfg.OpenAI.Model)
		reg.Register(oai)
		reg.SetVoice(oai)
		log.Println("Registered OpenAI provider")
	}
	if cfg.Gemini.Available {
		gem := provider.NewGemini(cfg.Gemini.APIKey, cfg.Gemini.Model)
		reg.Register(gem)
		log.Println("Registered Gemini provider")
	}

	if cfg.DefaultProvider != "" && cfg.DefaultProvider != reg.ActiveName() {
		if err := reg.SetActive(cfg.DefaultProvider); err != nil {
			log.Printf("Could not set default provider %q: %v", cfg.DefaultProvider, err)
		}
	}

	chatCfg := handler.ChatConfig{
		SystemPrompt: cfg.SystemPrompt,
		MaxMessages:  cfg.MaxMessages,
	}

	healthH := handler.NewHealthHandler(reg)
	chatH := handler.NewChatHandler(store, reg, chatCfg)
	providerH := handler.NewProviderHandler(reg)
	convH := handler.NewConversationHandler(store)
	wsH := handler.NewWSHandler(store, reg, chatCfg)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))

	// API routes.
	r.Get("/health", healthH.ServeHTTP)
	r.Post("/api/chat", chatH.HandleChat)
	r.Post("/api/chat/stream", chatH.HandleStream)
	r.Get("/api/providers", providerH.HandleList)
	r.Put("/api/providers/active", providerH.HandleSetActive)
	r.Get("/api/conversation/{id}", convH.HandleGet)
	r.Get("/ws", wsH.ServeHTTP)

	// Static files.
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	log.Printf("Starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
