package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/trap-smtp/handlers"
	"github.com/probitas-test/state-servers/trap-smtp/store"
)

func setupCountRouter() (*chi.Mux, *store.Store) {
	s := store.New(100, 0)
	handlers.SetStore(s)

	r := chi.NewRouter()
	r.Get("/api/count", handlers.CountHandler)
	return r, s
}

func TestCountHandler_Empty(t *testing.T) {
	r, _ := setupCountRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/count", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["count"].(float64) != 0 {
		t.Errorf("count = %v, want 0", resp["count"])
	}
}

func TestCountHandler_NoFilter(t *testing.T) {
	r, s := setupCountRouter()

	s.Add(&store.EmailEntry{From: "a@example.com"})
	s.Add(&store.EmailEntry{From: "b@example.com"})
	s.Add(&store.EmailEntry{From: "c@example.com"})

	req := httptest.NewRequest(http.MethodGet, "/api/count", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["count"].(float64) != 3 {
		t.Errorf("count = %v, want 3", resp["count"])
	}
}

func TestCountHandler_WithFilter(t *testing.T) {
	r, s := setupCountRouter()

	s.Add(&store.EmailEntry{From: "alice@example.com", Subject: "Hello"})
	s.Add(&store.EmailEntry{From: "bob@example.com", Subject: "Hello"})
	s.Add(&store.EmailEntry{From: "alice@example.com", Subject: "Goodbye"})

	req := httptest.NewRequest(http.MethodGet, "/api/count?from=alice", nil)
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

func TestCountHandler_MultipleFilters(t *testing.T) {
	r, s := setupCountRouter()

	s.Add(&store.EmailEntry{From: "alice@example.com", Subject: "Hello"})
	s.Add(&store.EmailEntry{From: "alice@example.com", Subject: "Goodbye"})
	s.Add(&store.EmailEntry{From: "bob@example.com", Subject: "Hello"})

	req := httptest.NewRequest(http.MethodGet, "/api/count?from=alice&subject=Hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["count"].(float64) != 1 {
		t.Errorf("count = %v, want 1", resp["count"])
	}
}

func TestCountHandler_NoMatch(t *testing.T) {
	r, s := setupCountRouter()

	s.Add(&store.EmailEntry{From: "alice@example.com"})

	req := httptest.NewRequest(http.MethodGet, "/api/count?from=nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["count"].(float64) != 0 {
		t.Errorf("count = %v, want 0", resp["count"])
	}
}
