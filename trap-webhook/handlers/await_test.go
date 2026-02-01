package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/state-webhook/handlers"
	"github.com/probitas-test/state-servers/state-webhook/store"
)

func setupAwaitRouter() (*chi.Mux, *store.Store) {
	s := store.New(100, 0)
	handlers.SetStore(s)

	r := chi.NewRouter()
	r.Get("/api/await", handlers.AwaitHandler)
	return r, s
}

func TestAwaitHandler_ExistingEntries(t *testing.T) {
	r, s := setupAwaitRouter()

	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/test"})

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=1s&path=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.WebhookEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}

func TestAwaitHandler_WaitsForEntry(t *testing.T) {
	r, s := setupAwaitRouter()

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/test"})
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=2s&path=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.WebhookEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}

func TestAwaitHandler_Timeout(t *testing.T) {
	r, _ := setupAwaitRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=200ms&path=nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("status = %d, want %d", w.Code, http.StatusRequestTimeout)
	}
}

func TestAwaitHandler_WithFilter(t *testing.T) {
	r, s := setupAwaitRouter()

	// Add non-matching entry
	s.Add(&store.WebhookEntry{Method: "GET", Path: "/webhook/other"})

	// Add matching entry after delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/target"})
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=2s&method=POST&path=target", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.WebhookEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Path != "/webhook/target" {
		t.Errorf("Path = %q, want %q", entries[0].Path, "/webhook/target")
	}
}

func TestAwaitHandler_MultipleCount(t *testing.T) {
	r, s := setupAwaitRouter()

	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/test"})

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/test"})
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=2&timeout=2s&path=test", nil)
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

func TestAwaitHandler_DefaultCount(t *testing.T) {
	r, s := setupAwaitRouter()

	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/test"})

	// No count param - should default to 1
	req := httptest.NewRequest(http.MethodGet, "/api/await?timeout=1s&path=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.WebhookEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}
