package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/trap-smtp/handlers"
	"github.com/probitas-test/state-servers/trap-smtp/store"
)

func setupTestRouter() (*chi.Mux, *store.Store) {
	s := store.New(100, 0)
	handlers.SetStore(s)
	handlers.SetSMTPPort("2525")

	r := chi.NewRouter()
	r.Get("/", handlers.UIHandler)
	r.Get("/api/entries", handlers.ListEntriesHandler)
	r.Get("/api/entries/{id}", handlers.GetEntryHandler)
	r.Get("/api/entries/{id}/raw", handlers.GetRawEntryHandler)
	r.Delete("/api/entries/{id}", handlers.DeleteEntryHandler)
	r.Delete("/api/entries", handlers.ClearEntriesHandler)
	r.Get("/api/stats", handlers.StatsHandler)

	return r, s
}

func TestListEntriesHandler(t *testing.T) {
	r, s := setupTestRouter()

	s.Add(&store.EmailEntry{Subject: "Email 1"})
	s.Add(&store.EmailEntry{Subject: "Email 2"})

	req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
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

func TestGetEntryHandler(t *testing.T) {
	r, s := setupTestRouter()

	id := s.Add(&store.EmailEntry{
		From:    "sender@example.com",
		Subject: "Test Subject",
		Body:    "Test body",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/entries/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var entry store.EmailEntry
	if err := json.NewDecoder(w.Body).Decode(&entry); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if entry.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", entry.Subject, "Test Subject")
	}
	if entry.From != "sender@example.com" {
		t.Errorf("From = %q, want %q", entry.From, "sender@example.com")
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

func TestGetRawEntryHandler(t *testing.T) {
	r, s := setupTestRouter()

	rawEmail := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nBody content"
	id := s.Add(&store.EmailEntry{
		Subject:  "Test",
		RawEmail: rawEmail,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/entries/"+id+"/raw", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "message/rfc822" {
		t.Errorf("Content-Type = %q, want %q", contentType, "message/rfc822")
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != rawEmail {
		t.Errorf("body = %q, want %q", string(body), rawEmail)
	}
}

func TestDeleteEntryHandler(t *testing.T) {
	r, s := setupTestRouter()

	id := s.Add(&store.EmailEntry{Subject: "Test"})

	req := httptest.NewRequest(http.MethodDelete, "/api/entries/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	_, ok := s.Get(id)
	if ok {
		t.Error("expected entry to be deleted")
	}
}

func TestClearEntriesHandler(t *testing.T) {
	r, s := setupTestRouter()

	s.Add(&store.EmailEntry{Subject: "1"})
	s.Add(&store.EmailEntry{Subject: "2"})
	s.Add(&store.EmailEntry{Subject: "3"})

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

	s.Add(&store.EmailEntry{})
	s.Add(&store.EmailEntry{})

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

func TestListEntriesHandler_RegexFilter(t *testing.T) {
	r, s := setupTestRouter()

	s.Add(&store.EmailEntry{From: "alice@example.com", Subject: "Hello"})
	s.Add(&store.EmailEntry{From: "bob@test.com", Subject: "Welcome"})
	s.Add(&store.EmailEntry{From: "alice@other.com", Subject: "Re: Hello"})

	req := httptest.NewRequest(http.MethodGet, "/api/entries?from_regex=^alice@", nil)
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

func TestListEntriesHandler_InvalidRegex(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/entries?subject_regex=[invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUIHandler(t *testing.T) {
	r, _ := setupTestRouter()

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
	if !strings.Contains(string(body), "SMTP Receiver") {
		t.Error("expected HTML to contain 'SMTP Receiver'")
	}

	// Check that SMTP port is interpolated
	if !strings.Contains(string(body), "2525") {
		t.Error("expected HTML to contain SMTP port '2525'")
	}
}
