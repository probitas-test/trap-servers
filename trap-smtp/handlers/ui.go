package handlers

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed ui.html
var uiHTML string

var smtpPort string

// SetSMTPPort sets the SMTP port for display in the UI
func SetSMTPPort(port string) {
	smtpPort = port
}

// UIHandler serves the web interface
func UIHandler(w http.ResponseWriter, r *http.Request) {
	html := strings.ReplaceAll(uiHTML, "{{SMTP_PORT}}", smtpPort)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
