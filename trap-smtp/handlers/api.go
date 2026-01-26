package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/state-smtp/store"
)

var emailStore *store.Store

// SetStore sets the email store for handlers
func SetStore(s *store.Store) {
	emailStore = s
}

// ListEntriesHandler returns stored email entries with optional filtering
func ListEntriesHandler(w http.ResponseWriter, r *http.Request) {
	filter := parseEmailFilter(r)
	entries := emailStore.ListWithFilter(filter)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		http.Error(w, "Failed to encode entries", http.StatusInternalServerError)
	}
}

// parseEmailFilter extracts filter parameters from the request
func parseEmailFilter(r *http.Request) *store.EmailFilter {
	q := r.URL.Query()

	filter := &store.EmailFilter{
		From:          q.Get("from"),
		To:            q.Get("to"),
		Subject:       q.Get("subject"),
		Body:          q.Get("body"),
		JSONPath:      q.Get("jsonpath"),
		JSONPathValue: q.Get("jsonpath_value"),
		Header:        q.Get("header"),
		HeaderValue:   q.Get("header_value"),
	}

	// Parse time filters
	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = &t
		}
	}
	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = &t
		}
	}

	// Parse pagination
	if limit := q.Get("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			filter.Limit = n
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil && n >= 0 {
			filter.Offset = n
		}
	}

	return filter
}

// GetEntryHandler returns a specific entry by ID
func GetEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	entry, ok := emailStore.Get(id)
	if !ok {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		http.Error(w, "Failed to encode entry", http.StatusInternalServerError)
	}
}

// GetRawEntryHandler returns the raw email content
func GetRawEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	entry, ok := emailStore.Get(id)
	if !ok {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "message/rfc822")
	w.Header().Set("Content-Disposition", `attachment; filename="email.eml"`)
	_, _ = w.Write([]byte(entry.RawEmail))
}

// DeleteEntryHandler deletes a specific entry by ID
func DeleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !emailStore.Delete(id) {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"deleted"}`))
}

// ClearEntriesHandler deletes all entries
func ClearEntriesHandler(w http.ResponseWriter, r *http.Request) {
	count := emailStore.Clear()

	w.Header().Set("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]interface{}{
		"status":  "cleared",
		"deleted": count,
	})
	_, _ = w.Write(resp)
}

// StatsHandler returns store statistics
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]interface{}{
		"count": emailStore.Count(),
	})
	_, _ = w.Write(resp)
}
