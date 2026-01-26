package handlers

import (
	_ "embed"
	"net/http"
)

//go:embed doc/api.md
var apiDoc []byte

// DocHandler serves the API documentation
func DocHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write(apiDoc)
}
