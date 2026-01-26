package smtp

import (
	"bytes"
	"io"
	"net/mail"

	"github.com/emersion/go-smtp"

	"github.com/probitas-test/state-servers/state-smtp/store"
)

// Backend implements the SMTP server backend
type Backend struct {
	store *store.Store
}

// NewBackend creates a new SMTP backend
func NewBackend(s *store.Store) *Backend {
	return &Backend{store: s}
}

// NewSession creates a new SMTP session
func (b *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{backend: b}, nil
}

// Session represents an SMTP session
type Session struct {
	backend *Backend
	from    string
	to      []string
}

// AuthPlain handles PLAIN authentication (accepts any credentials)
func (s *Session) AuthPlain(username, password string) error {
	return nil
}

// Mail handles the MAIL FROM command
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

// Rcpt handles the RCPT TO command
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

// Data handles the DATA command and receives the email content
func (s *Session) Data(r io.Reader) error {
	// Read the entire email
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// Parse the email
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		// Store even if parsing fails
		entry := &store.EmailEntry{
			From:     s.from,
			To:       s.to,
			RawEmail: string(raw),
			Body:     string(raw),
		}
		s.backend.store.Add(entry)
		return nil
	}

	// Extract headers
	headers := make(map[string][]string)
	for k, v := range msg.Header {
		headers[k] = v
	}

	// Read body
	body, _ := io.ReadAll(msg.Body)

	// Create entry
	entry := &store.EmailEntry{
		From:        s.from,
		To:          s.to,
		Subject:     msg.Header.Get("Subject"),
		Date:        msg.Header.Get("Date"),
		MessageID:   msg.Header.Get("Message-ID"),
		ContentType: msg.Header.Get("Content-Type"),
		Headers:     headers,
		Body:        string(body),
		RawEmail:    string(raw),
	}

	s.backend.store.Add(entry)
	return nil
}

// Reset resets the session state
func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

// Logout handles session logout
func (s *Session) Logout() error {
	return nil
}
