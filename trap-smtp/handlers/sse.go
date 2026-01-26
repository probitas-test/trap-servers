package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEHandler provides Server-Sent Events for real-time email updates
func SSEHandler(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to new entries
	ch := emailStore.Subscribe()
	defer emailStore.Unsubscribe(ch)

	// Send initial connection event
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n"); err != nil {
		return
	}
	flusher.Flush()

	// Stream new entries
	for {
		select {
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(entry)
			if _, err := fmt.Fprintf(w, "event: email\ndata: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
