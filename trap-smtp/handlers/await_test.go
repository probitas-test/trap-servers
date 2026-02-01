package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/state-smtp/handlers"
	"github.com/probitas-test/state-servers/state-smtp/store"
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

	s.Add(&store.EmailEntry{From: "test@example.com", Subject: "Hello"})

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=1s&from=test@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.EmailEntry
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
		s.Add(&store.EmailEntry{From: "test@example.com"})
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=2s&from=test@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.EmailEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}

func TestAwaitHandler_Timeout(t *testing.T) {
	r, _ := setupAwaitRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=200ms&from=nonexistent@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("status = %d, want %d", w.Code, http.StatusRequestTimeout)
	}
}

func TestAwaitHandler_WithFilter(t *testing.T) {
	r, s := setupAwaitRouter()

	// Add non-matching entry
	s.Add(&store.EmailEntry{From: "other@example.com", Subject: "Other"})

	// Add matching entry after delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Add(&store.EmailEntry{From: "target@example.com", Subject: "Target"})
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=1&timeout=2s&from=target@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.EmailEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].From != "target@example.com" {
		t.Errorf("From = %q, want %q", entries[0].From, "target@example.com")
	}
}

func TestAwaitHandler_MultipleCount(t *testing.T) {
	r, s := setupAwaitRouter()

	s.Add(&store.EmailEntry{From: "test@example.com", Subject: "First"})

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Add(&store.EmailEntry{From: "test@example.com", Subject: "Second"})
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/await?count=2&timeout=2s&from=test@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.EmailEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(entries))
	}
}

func TestAwaitHandler_DefaultCount(t *testing.T) {
	r, s := setupAwaitRouter()

	s.Add(&store.EmailEntry{From: "test@example.com"})

	// No count param - should default to 1
	req := httptest.NewRequest(http.MethodGet, "/api/await?timeout=1s&from=test@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entries []*store.EmailEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}
