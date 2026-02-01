package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/state-webhook/store"
)

// ListEntriesHandler returns stored webhook entries with optional filtering
func ListEntriesHandler(w http.ResponseWriter, r *http.Request) {
	filter, err := parseWebhookFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entries := webhookStore.ListWithFilter(filter)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		http.Error(w, "Failed to encode entries", http.StatusInternalServerError)
	}
}

// parseWebhookFilter extracts filter parameters from the request
func parseWebhookFilter(r *http.Request) (*store.WebhookFilter, error) {
	q := r.URL.Query()

	filter := &store.WebhookFilter{
		Method:        q.Get("method"),
		Path:          q.Get("path"),
		Query:         q.Get("query"),
		Body:          q.Get("body"),
		JSONPath:      q.Get("jsonpath"),
		JSONPathValue: q.Get("jsonpath_value"),
		ContentType:   q.Get("content_type"),
		Header:        q.Get("header"),
		HeaderValue:   q.Get("header_value"),
		Host:          q.Get("host"),
	}

	// Parse regex filters
	regexFields := []struct {
		param string
		dest  **regexp.Regexp
	}{
		{"path_regex", &filter.PathRegex},
		{"query_regex", &filter.QueryRegex},
		{"body_regex", &filter.BodyRegex},
		{"content_type_regex", &filter.ContentTypeRegex},
		{"host_regex", &filter.HostRegex},
	}
	for _, rf := range regexFields {
		if v := q.Get(rf.param); v != "" {
			re, err := regexp.Compile(v)
			if err != nil {
				return nil, fmt.Errorf("invalid %s: %w", rf.param, err)
			}
			*rf.dest = re
		}
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

	return filter, nil
}

// GetEntryHandler returns a specific entry by ID
func GetEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	entry, ok := webhookStore.Get(id)
	if !ok {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		http.Error(w, "Failed to encode entry", http.StatusInternalServerError)
	}
}

// DeleteEntryHandler deletes a specific entry by ID
func DeleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !webhookStore.Delete(id) {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"deleted"}`))
}

// ClearEntriesHandler deletes all entries
func ClearEntriesHandler(w http.ResponseWriter, r *http.Request) {
	count := webhookStore.Clear()

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
		"count": webhookStore.Count(),
	})
	_, _ = w.Write(resp)
}
