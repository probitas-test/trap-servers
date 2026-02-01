package smtp

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"

	"github.com/probitas-test/state-servers/trap-smtp/store"
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

	// Extract and decode headers (RFC 2047)
	headers := make(map[string][]string)
	for k, v := range msg.Header {
		decodedValues := make([]string, len(v))
		for i, val := range v {
			decodedValues[i] = decodeRFC2047(val)
		}
		headers[k] = decodedValues
	}

	// Helper to get first decoded header value
	getHeader := func(key string) string {
		if vals, ok := headers[key]; ok && len(vals) > 0 {
			return vals[0]
		}
		return ""
	}

	// Read and parse body (handles multipart MIME)
	rawBody, _ := io.ReadAll(msg.Body)
	parsed := parseEmailBody(rawBody, getHeader("Content-Type"))

	// Determine primary body (prefer HTML over text)
	primaryBody := parsed.htmlBody
	if primaryBody == "" {
		primaryBody = parsed.textBody
	}

	// Create entry with decoded headers and parsed body
	entry := &store.EmailEntry{
		From:        s.from,
		To:          s.to,
		Subject:     getHeader("Subject"),
		Date:        getHeader("Date"),
		MessageID:   getHeader("Message-Id"),
		ContentType: parsed.contentType,
		Headers:     headers,
		Body:        primaryBody,
		HtmlBody:    parsed.htmlBody,
		TextBody:    parsed.textBody,
		RawEmail:    string(raw),
		Attachments: parsed.attachments,
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

// decodeRFC2047 decodes RFC 2047 encoded-word headers (e.g., =?UTF-8?B?...?=)
// Returns the original string if decoding fails
func decodeRFC2047(s string) string {
	decoder := &mime.WordDecoder{}
	decoded, err := decoder.DecodeHeader(s)
	if err != nil {
		return s // Return original if decoding fails
	}
	return decoded
}

// parsedBody holds the result of MIME body parsing
type parsedBody struct {
	htmlBody    string
	textBody    string
	contentType string // Content-Type of the primary body (HTML if available)
	attachments []store.Attachment
}

// parseEmailBody extracts body content and attachments from an email
// For multipart emails, it extracts HTML, plain text, and attachments
func parseEmailBody(body []byte, contentType string) parsedBody {
	var result parsedBody
	parseMultipart(body, contentType, &result)
	return result
}

// parseMultipart recursively parses multipart MIME content
func parseMultipart(body []byte, contentType string, result *parsedBody) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		result.textBody = string(body)
		result.contentType = contentType
		return
	}

	// Handle multipart messages
	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			result.textBody = string(body)
			result.contentType = contentType
			return
		}

		reader := multipart.NewReader(bytes.NewReader(body), boundary)

		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}

			partContentType := part.Header.Get("Content-Type")
			partMediaType, partParams, _ := mime.ParseMediaType(partContentType)
			partBody, err := io.ReadAll(part)
			if err != nil {
				continue
			}

			// Handle nested multipart
			if strings.HasPrefix(partMediaType, "multipart/") {
				parseMultipart(partBody, partContentType, result)
				continue
			}

			// Check Content-Disposition
			disposition := part.Header.Get("Content-Disposition")
			contentID := strings.Trim(part.Header.Get("Content-Id"), "<>")

			isAttachment := strings.HasPrefix(disposition, "attachment")
			isInline := strings.HasPrefix(disposition, "inline") || contentID != ""

			if isAttachment || (isInline && !strings.HasPrefix(partMediaType, "text/")) {
				// This is an attachment or inline image
				decodedData := decodeTransferEncodingBytes(partBody, part.Header.Get("Content-Transfer-Encoding"))

				filename := getFilename(disposition, partParams)
				if filename == "" && contentID != "" {
					filename = contentID
				}

				attachment := store.Attachment{
					ID:          generateAttachmentID(),
					Filename:    filename,
					ContentType: partContentType,
					Size:        len(decodedData),
					Sha256:      fmt.Sprintf("%x", sha256.Sum256(decodedData)),
					ContentID:   contentID,
					IsInline:    isInline && contentID != "",
					Data:        decodedData,
				}
				result.attachments = append(result.attachments, attachment)
			} else if strings.HasPrefix(partMediaType, "text/html") {
				decoded := decodeTransferEncoding(partBody, part.Header.Get("Content-Transfer-Encoding"))
				result.htmlBody = decoded
				result.contentType = partContentType
			} else if strings.HasPrefix(partMediaType, "text/plain") {
				decoded := decodeTransferEncoding(partBody, part.Header.Get("Content-Transfer-Encoding"))
				result.textBody = decoded
				if result.contentType == "" {
					result.contentType = partContentType
				}
			}
		}
		return
	}

	// Single part message
	decoded := decodeTransferEncoding(body, "")
	if strings.HasPrefix(mediaType, "text/html") {
		result.htmlBody = decoded
		result.contentType = contentType
	} else {
		result.textBody = decoded
		result.contentType = contentType
	}
}

// getFilename extracts filename from Content-Disposition or Content-Type params
func getFilename(disposition string, params map[string]string) string {
	// Try Content-Disposition filename parameter
	_, dispParams, err := mime.ParseMediaType(disposition)
	if err == nil {
		if name := dispParams["filename"]; name != "" {
			return decodeRFC2047(name)
		}
	}
	// Try Content-Type name parameter
	if name := params["name"]; name != "" {
		return decodeRFC2047(name)
	}
	return ""
}

// generateAttachmentID creates a unique ID for attachments using UUID
func generateAttachmentID() string {
	return uuid.New().String()
}

// decodeTransferEncodingBytes decodes base64 or quoted-printable content to bytes
func decodeTransferEncodingBytes(body []byte, encoding string) []byte {
	encoding = strings.ToLower(strings.TrimSpace(encoding))

	switch encoding {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(body)))
		if err != nil {
			return body
		}
		return decoded
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			return body
		}
		return decoded
	default:
		return body
	}
}

// decodeTransferEncoding decodes base64 or quoted-printable content
func decodeTransferEncoding(body []byte, encoding string) string {
	encoding = strings.ToLower(strings.TrimSpace(encoding))

	switch encoding {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(body)))
		if err != nil {
			return string(body)
		}
		return string(decoded)
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			return string(body)
		}
		return string(decoded)
	default:
		return string(body)
	}
}
