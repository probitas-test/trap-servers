package store

import (
	"regexp"
	"testing"
	"time"
)

func TestWebhookFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter WebhookFilter
		want   bool
	}{
		{
			name:   "empty filter",
			filter: WebhookFilter{},
			want:   true,
		},
		{
			name:   "filter with method",
			filter: WebhookFilter{Method: "POST"},
			want:   false,
		},
		{
			name:   "filter with limit only (still empty for matching)",
			filter: WebhookFilter{Limit: 10},
			want:   true,
		},
		{
			name:   "filter with since",
			filter: WebhookFilter{Since: func() *time.Time { t := time.Now(); return &t }()},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_ListWithFilter(t *testing.T) {
	s := New(100, 0)

	// Add test entries
	now := time.Now()
	entries := []*WebhookEntry{
		{
			ID:          "1",
			ReceivedAt:  now.Add(-3 * time.Hour),
			Method:      "GET",
			Path:        "/api/users",
			QueryString: "page=1&limit=10",
			Body:        "",
			ContentType: "",
			Headers:     map[string][]string{"Accept": {"application/json"}},
			Host:        "api.example.com",
		},
		{
			ID:          "2",
			ReceivedAt:  now.Add(-2 * time.Hour),
			Method:      "POST",
			Path:        "/api/webhooks/events",
			QueryString: "token=abc123",
			Body:        `{"event":"user.created","data":{"id":456}}`,
			ContentType: "application/json",
			Headers:     map[string][]string{"Authorization": {"Bearer token123"}, "X-Custom": {"test"}},
			Host:        "hooks.example.com",
		},
		{
			ID:          "3",
			ReceivedAt:  now.Add(-1 * time.Hour),
			Method:      "POST",
			Path:        "/api/users",
			QueryString: "",
			Body:        `{"name":"John","email":"john@example.com"}`,
			ContentType: "application/json; charset=utf-8",
			Headers:     map[string][]string{"Authorization": {"Basic dXNlcjpwYXNz"}},
			Host:        "api.example.com",
		},
	}

	for _, e := range entries {
		s.Add(e)
	}

	tests := []struct {
		name      string
		filter    *WebhookFilter
		wantCount int
		wantIDs   []string // expected IDs in order (newest first)
	}{
		{
			name:      "empty filter returns all",
			filter:    &WebhookFilter{},
			wantCount: 3,
			wantIDs:   []string{"3", "2", "1"},
		},
		{
			name:      "filter by method (exact, case-insensitive)",
			filter:    &WebhookFilter{Method: "post"},
			wantCount: 2,
			wantIDs:   []string{"3", "2"},
		},
		{
			name:      "filter by path",
			filter:    &WebhookFilter{Path: "/api/users"},
			wantCount: 2,
			wantIDs:   []string{"3", "1"},
		},
		{
			name:      "filter by query string",
			filter:    &WebhookFilter{Query: "token"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by body",
			filter:    &WebhookFilter{Body: "john"},
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "filter by JSONPath existence",
			filter:    &WebhookFilter{JSONPath: "$.event"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by JSONPath with value",
			filter:    &WebhookFilter{JSONPath: "$.data.id", JSONPathValue: "456"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by JSONPath with wrong value",
			filter:    &WebhookFilter{JSONPath: "$.data.id", JSONPathValue: "789"},
			wantCount: 0,
		},
		{
			name:      "filter by content type",
			filter:    &WebhookFilter{ContentType: "json"},
			wantCount: 2,
			wantIDs:   []string{"3", "2"},
		},
		{
			name:      "filter by header existence",
			filter:    &WebhookFilter{Header: "Authorization"},
			wantCount: 2,
			wantIDs:   []string{"3", "2"},
		},
		{
			name:      "filter by header with value",
			filter:    &WebhookFilter{Header: "Authorization", HeaderValue: "Bearer"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by host",
			filter:    &WebhookFilter{Host: "api.example"},
			wantCount: 2,
			wantIDs:   []string{"3", "1"},
		},
		{
			name:      "filter by time range (since)",
			filter:    &WebhookFilter{Since: func() *time.Time { t := now.Add(-90 * time.Minute); return &t }()},
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "filter by time range (until)",
			filter:    &WebhookFilter{Until: func() *time.Time { t := now.Add(-90 * time.Minute); return &t }()},
			wantCount: 2,
			wantIDs:   []string{"2", "1"},
		},
		{
			name:      "filter with limit",
			filter:    &WebhookFilter{Limit: 2},
			wantCount: 2,
			wantIDs:   []string{"3", "2"},
		},
		{
			name:      "filter with offset",
			filter:    &WebhookFilter{Offset: 1},
			wantCount: 2,
			wantIDs:   []string{"2", "1"},
		},
		{
			name:      "filter with offset and limit",
			filter:    &WebhookFilter{Offset: 1, Limit: 1},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "combined filters (AND logic)",
			filter:    &WebhookFilter{Method: "POST", Host: "api.example"},
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "no matches",
			filter:    &WebhookFilter{Method: "DELETE"},
			wantCount: 0,
		},
		{
			name:      "filter by path regex",
			filter:    &WebhookFilter{PathRegex: regexp.MustCompile(`^/api/users$`)},
			wantCount: 2,
			wantIDs:   []string{"3", "1"},
		},
		{
			name:      "filter by path regex (webhooks only)",
			filter:    &WebhookFilter{PathRegex: regexp.MustCompile(`/webhooks/`)},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by body regex",
			filter:    &WebhookFilter{BodyRegex: regexp.MustCompile(`"event":"user\.\w+"`)},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by host regex",
			filter:    &WebhookFilter{HostRegex: regexp.MustCompile(`^(api|hooks)\.example\.com$`)},
			wantCount: 3,
			wantIDs:   []string{"3", "2", "1"},
		},
		{
			name:      "filter by host regex (specific)",
			filter:    &WebhookFilter{HostRegex: regexp.MustCompile(`^hooks\.`)},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "regex combined with contains filter (AND logic)",
			filter:    &WebhookFilter{Method: "POST", PathRegex: regexp.MustCompile(`/users$`)},
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "regex no match",
			filter:    &WebhookFilter{PathRegex: regexp.MustCompile(`^/nonexistent`)},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.ListWithFilter(tt.filter)
			if len(result) != tt.wantCount {
				t.Errorf("ListWithFilter() got %d entries, want %d", len(result), tt.wantCount)
			}
			if tt.wantIDs != nil {
				for i, id := range tt.wantIDs {
					if i >= len(result) {
						break
					}
					if result[i].ID != id {
						t.Errorf("ListWithFilter()[%d].ID = %s, want %s", i, result[i].ID, id)
					}
				}
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			if got := containsIgnoreCase(tt.s, tt.substr); got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestMatchJSONPath(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		path  string
		value string
		want  bool
	}{
		{
			name: "simple path match",
			body: `{"name":"John"}`,
			path: "$.name",
			want: true,
		},
		{
			name:  "path with value match",
			body:  `{"name":"John"}`,
			path:  "$.name",
			value: "John",
			want:  true,
		},
		{
			name:  "path with wrong value",
			body:  `{"name":"John"}`,
			path:  "$.name",
			value: "Jane",
			want:  false,
		},
		{
			name: "nested path",
			body: `{"user":{"id":123,"name":"John"}}`,
			path: "$.user.id",
			want: true,
		},
		{
			name:  "nested path with numeric value",
			body:  `{"user":{"id":123}}`,
			path:  "$.user.id",
			value: "123",
			want:  true,
		},
		{
			name: "path not found",
			body: `{"name":"John"}`,
			path: "$.age",
			want: false,
		},
		{
			name: "invalid JSON",
			body: "not json",
			path: "$.name",
			want: false,
		},
		{
			name: "array access",
			body: `{"items":[1,2,3]}`,
			path: "$.items[0]",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchJSONPath(tt.body, tt.path, tt.value); got != tt.want {
				t.Errorf("matchJSONPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchHeader(t *testing.T) {
	headers := map[string][]string{
		"Content-Type":    {"application/json"},
		"X-Custom-Header": {"value1", "value2"},
	}

	tests := []struct {
		name  string
		hname string
		value string
		want  bool
	}{
		{"header exists", "Content-Type", "", true},
		{"header exists case insensitive", "content-type", "", true},
		{"header with value", "Content-Type", "json", true},
		{"header with wrong value", "Content-Type", "xml", false},
		{"header not found", "X-Missing", "", false},
		{"multi-value header", "X-Custom-Header", "value2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchHeader(headers, tt.hname, tt.value); got != tt.want {
				t.Errorf("matchHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyPagination(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	tests := []struct {
		name   string
		offset int
		limit  int
		want   []int
	}{
		{"no pagination", 0, 0, []int{1, 2, 3, 4, 5}},
		{"limit only", 0, 3, []int{1, 2, 3}},
		{"offset only", 2, 0, []int{3, 4, 5}},
		{"offset and limit", 1, 2, []int{2, 3}},
		{"offset beyond length", 10, 0, []int{}},
		{"limit beyond remaining", 3, 10, []int{4, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyPagination(items, tt.offset, tt.limit)
			if len(got) != len(tt.want) {
				t.Errorf("applyPagination() got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("applyPagination()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}
