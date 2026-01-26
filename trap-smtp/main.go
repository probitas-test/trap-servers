package main

import (
	"log"
	"net/http"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/probitas-test/state-servers/state-smtp/handlers"
	smtpbackend "github.com/probitas-test/state-servers/state-smtp/smtp"
	"github.com/probitas-test/state-servers/state-smtp/store"
)

func main() {
	cfg := LoadConfig()

	// Initialize store
	emailStore := store.New(cfg.MaxEntries, cfg.EntryTTL)
	handlers.SetStore(emailStore)
	handlers.SetSMTPPort(cfg.SMTPPort)

	// Start SMTP server in goroutine
	go startSMTPServer(cfg, emailStore)

	// Start HTTP server
	startHTTPServer(cfg)
}

func startSMTPServer(cfg *Config, emailStore *store.Store) {
	backend := smtpbackend.NewBackend(emailStore)

	s := smtp.NewServer(backend)
	s.Addr = cfg.SMTPAddr()
	s.Domain = cfg.SMTPDomain
	s.WriteTimeout = 10 * time.Second
	s.ReadTimeout = 10 * time.Second
	s.MaxMessageBytes = int64(cfg.SMTPMaxSize)
	s.MaxRecipients = 50
	s.AllowInsecureAuth = cfg.AllowInsecure

	log.Printf("Starting SMTP server on %s", cfg.SMTPAddr())
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start SMTP server: %v", err)
	}
}

func startHTTPServer(cfg *Config) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Web UI
	r.Get("/", handlers.UIHandler)

	// API endpoints
	r.Route("/api", func(r chi.Router) {
		r.Get("/entries", handlers.ListEntriesHandler)
		r.Get("/entries/{id}", handlers.GetEntryHandler)
		r.Get("/entries/{id}/raw", handlers.GetRawEntryHandler)
		r.Delete("/entries/{id}", handlers.DeleteEntryHandler)
		r.Delete("/entries", handlers.ClearEntriesHandler)
		r.Get("/stats", handlers.StatsHandler)
		r.Get("/events", handlers.SSEHandler)
	})

	// API documentation
	r.Get("/doc", handlers.DocHandler)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("Starting HTTP server on %s", cfg.WebAddr())
	log.Printf("Web UI: http://%s/", cfg.WebAddr())
	if err := http.ListenAndServe(cfg.WebAddr(), r); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
