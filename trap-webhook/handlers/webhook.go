package handlers

import (
	"io"
	"net/http"

	"github.com/probitas-test/state-servers/state-webhook/store"
)

var webhookStore *store.Store

// SetStore sets the webhook store for handlers
func SetStore(s *store.Store) {
	webhookStore = s
}

// WebhookHandler accepts any HTTP request and stores it
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	// Create entry
	entry := &store.WebhookEntry{
		Method:      r.Method,
		Path:        r.URL.Path,
		QueryString: r.URL.RawQuery,
		Headers:     r.Header,
		Body:        string(body),
		ContentType: r.Header.Get("Content-Type"),
		RemoteAddr:  r.RemoteAddr,
		Host:        r.Host,
	}

	// Store entry
	id := webhookStore.Add(entry)

	// Return the entry ID
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Webhook-ID", id)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"id":"` + id + `","status":"received"}`))
}
