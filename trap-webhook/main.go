package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/probitas-test/state-servers/state-webhook/handlers"
	"github.com/probitas-test/state-servers/state-webhook/store"
)

func main() {
	cfg := LoadConfig()

	// Initialize store
	webhookStore := store.New(cfg.MaxEntries, cfg.EntryTTL)
	handlers.SetStore(webhookStore)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Web UI
	r.Get("/", handlers.UIHandler)

	// API endpoints
	r.Route("/api", func(r chi.Router) {
		r.Get("/entries", handlers.ListEntriesHandler)
		r.Get("/entries/{id}", handlers.GetEntryHandler)
		r.Delete("/entries/{id}", handlers.DeleteEntryHandler)
		r.Delete("/entries", handlers.ClearEntriesHandler)
		r.Get("/stats", handlers.StatsHandler)
		r.Get("/events", handlers.SSEHandler)
	})

	// Webhook receiver - accepts any method on any path under /webhook/
	r.HandleFunc("/webhook", handlers.WebhookHandler)
	r.HandleFunc("/webhook/*", handlers.WebhookHandler)

	// API documentation
	r.Get("/doc", handlers.DocHandler)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("Starting webhook receiver on %s", cfg.Addr())
	log.Printf("Web UI: http://%s/", cfg.Addr())
	log.Printf("Webhook URL: http://%s/webhook/", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), r); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
