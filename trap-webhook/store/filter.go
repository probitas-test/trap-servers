package store

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
)

// WebhookFilter defines criteria for filtering webhook entries
type WebhookFilter struct {
	Method        string     // Exact match on HTTP method
	Path          string     // Contains match on path
	Query         string     // Contains match on query string
	Body          string     // Contains match on body
	JSONPath      string     // JSONPath expression for body
	JSONPathValue string     // Expected value at JSONPath
	ContentType   string     // Contains match on content type
	Header        string     // Header name to check
	HeaderValue   string     // Header value contains
	Host          string     // Contains match on host
	Since         *time.Time // ReceivedAt after
	Until         *time.Time // ReceivedAt before
	Limit         int        // Max results (0 = no limit)
	Offset        int        // Skip first N results

	// Regex filters (compiled regular expressions)
	PathRegex        *regexp.Regexp // Regex match on path
	QueryRegex       *regexp.Regexp // Regex match on query string
	BodyRegex        *regexp.Regexp // Regex match on body
	ContentTypeRegex *regexp.Regexp // Regex match on content type
	HostRegex        *regexp.Regexp // Regex match on host
}

// IsEmpty returns true if no filter criteria are set
func (f *WebhookFilter) IsEmpty() bool {
	return f.Method == "" &&
		f.Path == "" &&
		f.Query == "" &&
		f.Body == "" &&
		f.JSONPath == "" &&
		f.ContentType == "" &&
		f.Header == "" &&
		f.Host == "" &&
		f.Since == nil &&
		f.Until == nil &&
		f.PathRegex == nil &&
		f.QueryRegex == nil &&
		f.BodyRegex == nil &&
		f.ContentTypeRegex == nil &&
		f.HostRegex == nil
}

// ListWithFilter returns entries matching the filter criteria
func (s *Store) ListWithFilter(filter *WebhookFilter) []*WebhookEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start with all entries in reverse chronological order (newest first)
	var result []*WebhookEntry
	for i := len(s.order) - 1; i >= 0; i-- {
		entry := s.entries[s.order[i]]
		if matchesWebhookFilter(entry, filter) {
			result = append(result, entry)
		}
	}

	// Apply pagination
	return applyPagination(result, filter.Offset, filter.Limit)
}

// matchesWebhookFilter checks if an entry matches all filter criteria (AND logic)
func matchesWebhookFilter(entry *WebhookEntry, filter *WebhookFilter) bool {
	// Method filter (exact match, case-insensitive)
	if filter.Method != "" && !strings.EqualFold(entry.Method, filter.Method) {
		return false
	}

	// Path filter
	if filter.Path != "" && !containsIgnoreCase(entry.Path, filter.Path) {
		return false
	}
	if filter.PathRegex != nil && !filter.PathRegex.MatchString(entry.Path) {
		return false
	}

	// Query string filter
	if filter.Query != "" && !containsIgnoreCase(entry.QueryString, filter.Query) {
		return false
	}
	if filter.QueryRegex != nil && !filter.QueryRegex.MatchString(entry.QueryString) {
		return false
	}

	// Body filter
	if filter.Body != "" && !containsIgnoreCase(entry.Body, filter.Body) {
		return false
	}
	if filter.BodyRegex != nil && !filter.BodyRegex.MatchString(entry.Body) {
		return false
	}

	// JSONPath filter
	if filter.JSONPath != "" {
		if !matchJSONPath(entry.Body, filter.JSONPath, filter.JSONPathValue) {
			return false
		}
	}

	// Content-Type filter
	if filter.ContentType != "" && !containsIgnoreCase(entry.ContentType, filter.ContentType) {
		return false
	}
	if filter.ContentTypeRegex != nil && !filter.ContentTypeRegex.MatchString(entry.ContentType) {
		return false
	}

	// Header filter
	if filter.Header != "" {
		if !matchHeader(entry.Headers, filter.Header, filter.HeaderValue) {
			return false
		}
	}

	// Host filter
	if filter.Host != "" && !containsIgnoreCase(entry.Host, filter.Host) {
		return false
	}
	if filter.HostRegex != nil && !filter.HostRegex.MatchString(entry.Host) {
		return false
	}

	// Time range filter
	if !matchTimeRange(entry.ReceivedAt, filter.Since, filter.Until) {
		return false
	}

	return true
}

// containsIgnoreCase performs case-insensitive substring match
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// matchJSONPath evaluates a JSONPath expression against JSON content
func matchJSONPath(body, path, expectedValue string) bool {
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return false // Not valid JSON
	}

	result, err := jsonpath.Get(path, data)
	if err != nil {
		return false // JSONPath not found
	}

	// If no expected value specified, just check existence
	if expectedValue == "" {
		return true
	}

	// Compare values (convert to string for comparison)
	resultStr := fmt.Sprintf("%v", result)
	return resultStr == expectedValue
}

// matchHeader checks if a header exists and optionally matches a value
func matchHeader(headers map[string][]string, name, value string) bool {
	// Header names are case-insensitive in HTTP
	for headerName, values := range headers {
		if strings.EqualFold(headerName, name) {
			if value == "" {
				return true // Header exists, no value check needed
			}
			for _, v := range values {
				if containsIgnoreCase(v, value) {
					return true
				}
			}
		}
	}
	return false
}

// matchTimeRange checks if a time is within the specified range
func matchTimeRange(t time.Time, since, until *time.Time) bool {
	if since != nil && t.Before(*since) {
		return false
	}
	if until != nil && t.After(*until) {
		return false
	}
	return true
}

// applyPagination applies offset and limit to results
func applyPagination[T any](items []T, offset, limit int) []T {
	if offset > len(items) {
		return []T{}
	}

	items = items[offset:]

	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}

	return items
}
