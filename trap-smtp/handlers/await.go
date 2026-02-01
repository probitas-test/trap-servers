package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/probitas-test/state-servers/trap-smtp/store"
)

// AwaitHandler blocks until the specified number of entries match the filter,
// or the timeout is reached. Returns matched entries on success, 408 on timeout.
//
// Query parameters:
//   - count: minimum number of matching entries to wait for (default: 1)
//   - timeout: maximum wait duration, e.g. "5s", "500ms" (default: 10s)
//   - All email filter parameters (from, to, subject, body, etc.)
func AwaitHandler(w http.ResponseWriter, r *http.Request) {
	filter, err := parseAwaitFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	count := 1
	if c := r.URL.Query().Get("count"); c != "" {
		n, err := strconv.Atoi(c)
		if err != nil || n <= 0 {
			http.Error(w, "invalid 'count' parameter: must be a positive integer", http.StatusBadRequest)
			return
		}
		count = n
	}

	timeout := 10 * time.Second
	if t := r.URL.Query().Get("timeout"); t != "" {
		d, err := time.ParseDuration(t)
		if err != nil || d <= 0 {
			http.Error(w, "invalid 'timeout' parameter: must be a positive duration (e.g. '5s', '500ms')", http.StatusBadRequest)
			return
		}
		timeout = d
	}

	// Subscribe before checking existing entries to avoid missing entries
	// added between the check and subscribe.
	ch := emailStore.Subscribe()
	defer emailStore.Unsubscribe(ch)

	// Check existing entries
	entries := emailStore.ListWithFilter(filter)
	if len(entries) >= count {
		resp, err := json.Marshal(entries)
		if err != nil {
			http.Error(w, "Failed to encode entries", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(resp); err != nil {
			// Unable to write response; nothing more we can do here.
			return
		}
		return
	}

	// Wait for new entries
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				http.Error(w, "Store closed", http.StatusInternalServerError)
				return
			}
			entries = emailStore.ListWithFilter(filter)
			if len(entries) >= count {
				resp, err := json.Marshal(entries)
				if err != nil {
					http.Error(w, "Failed to encode entries", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write(resp); err != nil {
					// Unable to write response; nothing more we can do here.
					return
				}
				return
			}
		case <-timer.C:
			resp, err := json.Marshal(map[string]interface{}{
				"error":    "timeout",
				"matched":  len(emailStore.ListWithFilter(filter)),
				"expected": count,
			})
			if err != nil {
				http.Error(w, "Failed to encode timeout response", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			_, _ = w.Write(resp)
			return
		case <-r.Context().Done():
			return
		}
	}
}

// parseAwaitFilter extracts filter parameters without pagination (limit/offset).
func parseAwaitFilter(r *http.Request) (*store.EmailFilter, error) {
	filter, err := parseEmailFilter(r)
	if err != nil {
		return nil, err
	}
	// Clear pagination fields (not supported by await)
	filter.Limit = 0
	filter.Offset = 0
	return filter, nil
}
