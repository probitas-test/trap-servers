package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/state-webhook/handlers"
	"github.com/probitas-test/state-servers/state-webhook/store"
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

	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/a"})
	s.Add(&store.WebhookEntry{Method: "GET", Path: "/webhook/b"})
	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/c"})

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

	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/a"})
	s.Add(&store.WebhookEntry{Method: "GET", Path: "/webhook/b"})
	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/c"})

	req := httptest.NewRequest(http.MethodGet, "/api/count?method=POST", nil)
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

	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/payment"})
	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/notification"})
	s.Add(&store.WebhookEntry{Method: "GET", Path: "/webhook/payment"})

	req := httptest.NewRequest(http.MethodGet, "/api/count?method=POST&path=payment", nil)
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

	s.Add(&store.WebhookEntry{Method: "POST", Path: "/webhook/test"})

	req := httptest.NewRequest(http.MethodGet, "/api/count?method=DELETE", nil)
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
