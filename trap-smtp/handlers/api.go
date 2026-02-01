package handlers

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/probitas-test/state-servers/trap-smtp/store"
)

var emailStore *store.Store

// SetStore sets the email store for handlers
func SetStore(s *store.Store) {
	emailStore = s
}

// ListEntriesHandler returns stored email entries with optional filtering
func ListEntriesHandler(w http.ResponseWriter, r *http.Request) {
	filter, err := parseEmailFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entries := emailStore.ListWithFilter(filter)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		http.Error(w, "Failed to encode entries", http.StatusInternalServerError)
	}
}

// parseEmailFilter extracts filter parameters from the request
func parseEmailFilter(r *http.Request) (*store.EmailFilter, error) {
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

	// Parse regex filters
	regexFields := []struct {
		param string
		dest  **regexp.Regexp
	}{
		{"from_regex", &filter.FromRegex},
		{"to_regex", &filter.ToRegex},
		{"subject_regex", &filter.SubjectRegex},
		{"body_regex", &filter.BodyRegex},
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

// GetAttachmentHandler returns an attachment by entry ID and attachment ID
func GetAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "id")
	attachmentID := chi.URLParam(r, "attachmentId")

	entry, ok := emailStore.Get(entryID)
	if !ok {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	// Find attachment
	for _, att := range entry.Attachments {
		if att.ID == attachmentID {
			// Determine content type
			contentType := att.ContentType
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			w.Header().Set("Content-Type", contentType)
			if att.Filename != "" {
				// Use mime.FormatMediaType to safely encode filename
				disposition := mime.FormatMediaType("attachment", map[string]string{
					"filename": att.Filename,
				})
				w.Header().Set("Content-Disposition", disposition)
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(att.Data)))
			_, _ = w.Write(att.Data)
			return
		}
	}

	http.Error(w, "Attachment not found", http.StatusNotFound)
}

// allowedInlineContentTypes defines safe content types for inline display
var allowedInlineContentTypes = []string{
	"image/png", "image/jpeg", "image/gif", "image/webp", "image/svg+xml", "image/bmp",
}

// isAllowedInlineType checks if content type is safe for inline display
func isAllowedInlineType(contentType string) bool {
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	for _, allowed := range allowedInlineContentTypes {
		if ct == allowed {
			return true
		}
	}
	return false
}

// GetAttachmentByCIDHandler returns an inline attachment by Content-ID (for cid: references)
func GetAttachmentByCIDHandler(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "id")
	cid := chi.URLParam(r, "cid")

	entry, ok := emailStore.Get(entryID)
	if !ok {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	// Find attachment by Content-ID
	for _, att := range entry.Attachments {
		if att.ContentID == cid {
			contentType := att.ContentType
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			// Only allow safe image types for inline content
			if !isAllowedInlineType(contentType) {
				http.Error(w, "Content type not allowed for inline display", http.StatusForbidden)
				return
			}

			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "max-age=3600")
			_, _ = w.Write(att.Data)
			return
		}
	}

	http.Error(w, "Attachment not found", http.StatusNotFound)
}
