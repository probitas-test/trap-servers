package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/state-webhook/handlers"
	"github.com/probitas-test/state-servers/state-webhook/store"
)

func setupTestRouter() (*chi.Mux, *store.Store) {
	s := store.New(100, 0)
	handlers.SetStore(s)

	r := chi.NewRouter()
	r.HandleFunc("/webhook", handlers.WebhookHandler)
	r.HandleFunc("/webhook/*", handlers.WebhookHandler)
	r.Get("/api/entries", handlers.ListEntriesHandler)
	r.Get("/api/entries/{id}", handlers.GetEntryHandler)
	r.Delete("/api/entries/{id}", handlers.DeleteEntryHandler)
	r.Delete("/api/entries", handlers.ClearEntriesHandler)
	r.Get("/api/stats", handlers.StatsHandler)

	return r, s
}

func TestWebhookHandler_POST(t *testing.T) {
	r, s := setupTestRouter()

	body := `{"message": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/test-path", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Check response
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "received" {
		t.Errorf("status = %q, want %q", resp["status"], "received")
	}

	id := resp["id"]
	if id == "" {
		t.Error("expected non-empty id in response")
	}

	// Verify stored entry
	entry, ok := s.Get(id)
	if !ok {
		t.Fatal("entry not found in store")
	}

	if entry.Method != "POST" {
		t.Errorf("Method = %q, want %q", entry.Method, "POST")
	}
	if entry.Path != "/webhook/test-path" {
		t.Errorf("Path = %q, want %q", entry.Path, "/webhook/test-path")
	}
	if entry.Body != body {
		t.Errorf("Body = %q, want %q", entry.Body, body)
	}
	if v := entry.Headers["X-Custom-Header"]; len(v) == 0 || v[0] != "custom-value" {
		t.Errorf("X-Custom-Header = %v, want [custom-value]", v)
	}
}

func TestWebhookHandler_GET(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/webhook?param=value", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListEntriesHandler(t *testing.T) {
	r, s := setupTestRouter()

	// Add some entries
	s.Add(&store.WebhookEntry{Path: "/1"})
	s.Add(&store.WebhookEntry{Path: "/2"})

	req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.WebhookEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(entries))
	}
}

func TestGetEntryHandler(t *testing.T) {
	r, s := setupTestRouter()

	id := s.Add(&store.WebhookEntry{Path: "/test", Body: "test body"})

	req := httptest.NewRequest(http.MethodGet, "/api/entries/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entry store.WebhookEntry
	if err := json.NewDecoder(w.Body).Decode(&entry); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if entry.Path != "/test" {
		t.Errorf("Path = %q, want %q", entry.Path, "/test")
	}
}

func TestGetEntryHandler_NotFound(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/entries/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteEntryHandler(t *testing.T) {
	r, s := setupTestRouter()

	id := s.Add(&store.WebhookEntry{Path: "/test"})

	req := httptest.NewRequest(http.MethodDelete, "/api/entries/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify deleted
	_, ok := s.Get(id)
	if ok {
		t.Error("expected entry to be deleted")
	}
}

func TestClearEntriesHandler(t *testing.T) {
	r, s := setupTestRouter()

	s.Add(&store.WebhookEntry{Path: "/1"})
	s.Add(&store.WebhookEntry{Path: "/2"})
	s.Add(&store.WebhookEntry{Path: "/3"})

	req := httptest.NewRequest(http.MethodDelete, "/api/entries", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["deleted"].(float64) != 3 {
		t.Errorf("deleted = %v, want 3", resp["deleted"])
	}

	if s.Count() != 0 {
		t.Errorf("Count() = %d, want 0", s.Count())
	}
}

func TestStatsHandler(t *testing.T) {
	r, s := setupTestRouter()

	s.Add(&store.WebhookEntry{})
	s.Add(&store.WebhookEntry{})

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["count"].(float64) != 2 {
		t.Errorf("count = %v, want 2", resp["count"])
	}
}

func TestUIHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/", handlers.UIHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "Webhook Receiver") {
		t.Error("expected HTML to contain 'Webhook Receiver'")
	}
}
