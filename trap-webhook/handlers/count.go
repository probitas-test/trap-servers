package handlers

import (
	"encoding/json"
	"net/http"
)

// CountHandler returns the number of entries matching the filter criteria.
func CountHandler(w http.ResponseWriter, r *http.Request) {
	filter, err := parseWebhookFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Ignore pagination for counting
	filter.Limit = 0
	filter.Offset = 0

	entries := webhookStore.ListWithFilter(filter)

	resp, err := json.Marshal(map[string]interface{}{
		"count": len(entries),
	})
	if err != nil {
		http.Error(w, "failed to marshal count response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		// Unable to write response; nothing more we can do here.
		return
	}
}
