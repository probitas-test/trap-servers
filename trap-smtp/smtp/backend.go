package smtp

import (
	"bytes"
	"errors"
	"io"
	"net/mail"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"github.com/probitas-test/state-servers/state-smtp/store"
)

// AuthConfig holds authentication settings
type AuthConfig struct {
	Username string
	Password string
}

// HasCredentials returns true if authentication is configured
func (c *AuthConfig) HasCredentials() bool {
	return c.Username != "" && c.Password != ""
}

// Backend implements the SMTP server backend
type Backend struct {
	store *store.Store
	auth  *AuthConfig
}

// NewBackend creates a new SMTP backend
func NewBackend(s *store.Store, auth *AuthConfig) *Backend {
	return &Backend{store: s, auth: auth}
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

// AuthMechanisms returns the supported authentication mechanisms
// Always advertise PLAIN and LOGIN so clients can authenticate even in open relay mode
func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain, sasl.Login}
}

// Auth handles the AUTH command and returns a SASL server for the given mechanism
func (s *Session) Auth(mech string) (sasl.Server, error) {
	// If no credentials configured, accept any authentication (open relay mode)
	if !s.backend.auth.HasCredentials() {
		switch mech {
		case sasl.Plain:
			return sasl.NewPlainServer(func(identity, username, password string) error {
				return nil // Accept any credentials
			}), nil
		case sasl.Login:
			return &loginServer{acceptAny: true}, nil
		default:
			return nil, smtp.ErrAuthUnknownMechanism
		}
	}

	// Credentials configured, validate against them
	switch mech {
	case sasl.Plain:
		return sasl.NewPlainServer(func(identity, username, password string) error {
			if username == s.backend.auth.Username && password == s.backend.auth.Password {
				return nil
			}
			return errors.New("invalid credentials")
		}), nil
	case sasl.Login:
		return &loginServer{
			username: s.backend.auth.Username,
			password: s.backend.auth.Password,
		}, nil
	default:
		return nil, smtp.ErrAuthUnknownMechanism
	}
}

// loginServer implements sasl.Server for the LOGIN mechanism
type loginServer struct {
	username  string
	password  string
	acceptAny bool // If true, accept any credentials (open relay mode)
	step      int
	gotUser   string
}

func (s *loginServer) Next(response []byte) (challenge []byte, done bool, err error) {
	switch s.step {
	case 0:
		// Initial state: send username prompt
		s.step++
		return []byte("Username:"), false, nil
	case 1:
		// Got username, send password prompt
		s.gotUser = string(response)
		s.step++
		return []byte("Password:"), false, nil
	case 2:
		// Got password, verify credentials
		if s.acceptAny {
			return nil, true, nil // Accept any credentials in open relay mode
		}
		gotPass := string(response)
		if s.gotUser == s.username && gotPass == s.password {
			return nil, true, nil
		}
		return nil, true, errors.New("invalid credentials")
	default:
		return nil, true, errors.New("unexpected LOGIN state")
	}
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
