package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
)

// EmailFilter defines criteria for filtering email entries
type EmailFilter struct {
	From          string     // Contains match on sender
	To            string     // Contains match on any recipient
	Subject       string     // Contains match on subject
	Body          string     // Contains match on body
	JSONPath      string     // JSONPath expression for body
	JSONPathValue string     // Expected value at JSONPath
	Header        string     // Header name to check
	HeaderValue   string     // Header value contains
	Since         *time.Time // ReceivedAt after
	Until         *time.Time // ReceivedAt before
	Limit         int        // Max results (0 = no limit)
	Offset        int        // Skip first N results
}

// IsEmpty returns true if no filter criteria are set
func (f *EmailFilter) IsEmpty() bool {
	return f.From == "" &&
		f.To == "" &&
		f.Subject == "" &&
		f.Body == "" &&
		f.JSONPath == "" &&
		f.Header == "" &&
		f.Since == nil &&
		f.Until == nil
}

// ListWithFilter returns entries matching the filter criteria
func (s *Store) ListWithFilter(filter *EmailFilter) []*EmailEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start with all entries in reverse chronological order (newest first)
	var result []*EmailEntry
	for i := len(s.order) - 1; i >= 0; i-- {
		entry := s.entries[s.order[i]]
		if matchesEmailFilter(entry, filter) {
			result = append(result, entry)
		}
	}

	// Apply pagination
	return applyPagination(result, filter.Offset, filter.Limit)
}

// matchesEmailFilter checks if an entry matches all filter criteria (AND logic)
func matchesEmailFilter(entry *EmailEntry, filter *EmailFilter) bool {
	// From filter
	if filter.From != "" && !containsIgnoreCase(entry.From, filter.From) {
		return false
	}

	// To filter (any recipient must match)
	if filter.To != "" {
		found := false
		for _, to := range entry.To {
			if containsIgnoreCase(to, filter.To) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Subject filter
	if filter.Subject != "" && !containsIgnoreCase(entry.Subject, filter.Subject) {
		return false
	}

	// Body filter
	if filter.Body != "" && !containsIgnoreCase(entry.Body, filter.Body) {
		return false
	}

	// JSONPath filter
	if filter.JSONPath != "" {
		if !matchJSONPath(entry.Body, filter.JSONPath, filter.JSONPathValue) {
			return false
		}
	}

	// Header filter
	if filter.Header != "" {
		if !matchHeader(entry.Headers, filter.Header, filter.HeaderValue) {
			return false
		}
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
