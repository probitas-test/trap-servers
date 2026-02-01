package store

import (
	"regexp"
	"testing"
	"time"
)

func TestEmailFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter EmailFilter
		want   bool
	}{
		{
			name:   "empty filter",
			filter: EmailFilter{},
			want:   true,
		},
		{
			name:   "filter with from",
			filter: EmailFilter{From: "test@example.com"},
			want:   false,
		},
		{
			name:   "filter with limit only (still empty for matching)",
			filter: EmailFilter{Limit: 10},
			want:   true,
		},
		{
			name:   "filter with since",
			filter: EmailFilter{Since: func() *time.Time { t := time.Now(); return &t }()},
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
	entries := []*EmailEntry{
		{
			ID:         "1",
			ReceivedAt: now.Add(-3 * time.Hour),
			From:       "alice@example.com",
			To:         []string{"bob@example.com"},
			Subject:    "Hello World",
			Body:       "This is a test email",
			Headers:    map[string][]string{"X-Custom": {"value1"}},
		},
		{
			ID:         "2",
			ReceivedAt: now.Add(-2 * time.Hour),
			From:       "charlie@test.com",
			To:         []string{"admin@example.com", "support@example.com"},
			Subject:    "Welcome Message",
			Body:       `{"event":"created","user":{"id":123}}`,
			Headers:    map[string][]string{"X-Priority": {"high"}},
		},
		{
			ID:         "3",
			ReceivedAt: now.Add(-1 * time.Hour),
			From:       "bob@example.com",
			To:         []string{"alice@example.com"},
			Subject:    "Re: Hello World",
			Body:       "Thanks for the email",
			Headers:    map[string][]string{"X-Custom": {"value2"}, "X-Priority": {"low"}},
		},
	}

	for _, e := range entries {
		s.Add(e)
	}

	tests := []struct {
		name      string
		filter    *EmailFilter
		wantCount int
		wantIDs   []string // expected IDs in order (newest first)
	}{
		{
			name:      "empty filter returns all",
			filter:    &EmailFilter{},
			wantCount: 3,
			wantIDs:   []string{"3", "2", "1"},
		},
		{
			name:      "filter by from",
			filter:    &EmailFilter{From: "alice"},
			wantCount: 1,
			wantIDs:   []string{"1"},
		},
		{
			name:      "filter by to (any recipient)",
			filter:    &EmailFilter{To: "admin"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by subject",
			filter:    &EmailFilter{Subject: "hello"},
			wantCount: 2,
			wantIDs:   []string{"3", "1"},
		},
		{
			name:      "filter by body",
			filter:    &EmailFilter{Body: "email"},
			wantCount: 2,
			wantIDs:   []string{"3", "1"},
		},
		{
			name:      "filter by JSONPath existence",
			filter:    &EmailFilter{JSONPath: "$.event"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by JSONPath with value",
			filter:    &EmailFilter{JSONPath: "$.user.id", JSONPathValue: "123"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by JSONPath with wrong value",
			filter:    &EmailFilter{JSONPath: "$.user.id", JSONPathValue: "456"},
			wantCount: 0,
		},
		{
			name:      "filter by header existence",
			filter:    &EmailFilter{Header: "X-Custom"},
			wantCount: 2,
			wantIDs:   []string{"3", "1"},
		},
		{
			name:      "filter by header with value",
			filter:    &EmailFilter{Header: "X-Priority", HeaderValue: "high"},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by time range (since)",
			filter:    &EmailFilter{Since: func() *time.Time { t := now.Add(-90 * time.Minute); return &t }()},
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "filter by time range (until)",
			filter:    &EmailFilter{Until: func() *time.Time { t := now.Add(-90 * time.Minute); return &t }()},
			wantCount: 2,
			wantIDs:   []string{"2", "1"},
		},
		{
			name:      "filter with limit",
			filter:    &EmailFilter{Limit: 2},
			wantCount: 2,
			wantIDs:   []string{"3", "2"},
		},
		{
			name:      "filter with offset",
			filter:    &EmailFilter{Offset: 1},
			wantCount: 2,
			wantIDs:   []string{"2", "1"},
		},
		{
			name:      "filter with offset and limit",
			filter:    &EmailFilter{Offset: 1, Limit: 1},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "combined filters (AND logic)",
			filter:    &EmailFilter{From: "example.com", Subject: "hello"},
			wantCount: 2,
			wantIDs:   []string{"3", "1"},
		},
		{
			name:      "no matches",
			filter:    &EmailFilter{From: "nonexistent"},
			wantCount: 0,
		},
		{
			name:      "filter by from regex",
			filter:    &EmailFilter{FromRegex: regexp.MustCompile(`^alice@`)},
			wantCount: 1,
			wantIDs:   []string{"1"},
		},
		{
			name:      "filter by to regex (any recipient)",
			filter:    &EmailFilter{ToRegex: regexp.MustCompile(`(admin|support)@`)},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "filter by subject regex",
			filter:    &EmailFilter{SubjectRegex: regexp.MustCompile(`^Re:`)},
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "filter by body regex",
			filter:    &EmailFilter{BodyRegex: regexp.MustCompile(`"event":"\w+"`)},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "regex combined with contains filter (AND logic)",
			filter:    &EmailFilter{From: "example.com", SubjectRegex: regexp.MustCompile(`^Hello`)},
			wantCount: 1,
			wantIDs:   []string{"1"},
		},
		{
			name:      "regex no match",
			filter:    &EmailFilter{SubjectRegex: regexp.MustCompile(`^Goodbye`)},
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
