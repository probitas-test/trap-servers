package smtp_test

import (
	"strings"
	"testing"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"

	"github.com/probitas-test/state-servers/state-smtp/smtp"
	"github.com/probitas-test/state-servers/state-smtp/store"
)

// noAuth returns an AuthConfig with no credentials (open relay mode)
func noAuth() *smtp.AuthConfig {
	return &smtp.AuthConfig{}
}

// withAuth returns an AuthConfig with the given credentials
func withAuth(username, password string) *smtp.AuthConfig {
	return &smtp.AuthConfig{
		Username: username,
		Password: password,
	}
}

func TestBackend_NewSession(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, err := backend.NewSession(nil)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
}

func TestSession_FullFlow(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	// MAIL FROM
	err := session.Mail("sender@example.com", &gosmtp.MailOptions{})
	if err != nil {
		t.Fatalf("Mail() error = %v", err)
	}

	// RCPT TO
	err = session.Rcpt("recipient@example.com", &gosmtp.RcptOptions{})
	if err != nil {
		t.Fatalf("Rcpt() error = %v", err)
	}

	// DATA
	emailContent := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test Email\r\n\r\nThis is the email body."
	err = session.Data(strings.NewReader(emailContent))
	if err != nil {
		t.Fatalf("Data() error = %v", err)
	}

	// Verify email was stored
	if s.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", s.Count())
	}

	entries := s.List()
	entry := entries[0]

	if entry.From != "sender@example.com" {
		t.Errorf("From = %q, want %q", entry.From, "sender@example.com")
	}
	if len(entry.To) != 1 || entry.To[0] != "recipient@example.com" {
		t.Errorf("To = %v, want [recipient@example.com]", entry.To)
	}
	if entry.Subject != "Test Email" {
		t.Errorf("Subject = %q, want %q", entry.Subject, "Test Email")
	}
	if entry.Body != "This is the email body." {
		t.Errorf("Body = %q, want %q", entry.Body, "This is the email body.")
	}
	if entry.RawEmail != emailContent {
		t.Errorf("RawEmail = %q, want %q", entry.RawEmail, emailContent)
	}
}

func TestSession_MultipleRecipients(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	_ = session.Mail("sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("a@example.com", &gosmtp.RcptOptions{})
	_ = session.Rcpt("b@example.com", &gosmtp.RcptOptions{})
	_ = session.Rcpt("c@example.com", &gosmtp.RcptOptions{})

	emailContent := "Subject: Multi-recipient\r\n\r\nBody"
	_ = session.Data(strings.NewReader(emailContent))

	entries := s.List()
	if len(entries[0].To) != 3 {
		t.Errorf("len(To) = %d, want 3", len(entries[0].To))
	}
}

func TestSession_Reset(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	// Set up some state
	_ = session.Mail("sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("recipient@example.com", &gosmtp.RcptOptions{})

	// Reset
	session.Reset()

	// Send new email
	_ = session.Mail("new-sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("new-recipient@example.com", &gosmtp.RcptOptions{})
	_ = session.Data(strings.NewReader("Subject: After reset\r\n\r\nBody"))

	entries := s.List()
	if entries[0].From != "new-sender@example.com" {
		t.Errorf("From = %q, want %q", entries[0].From, "new-sender@example.com")
	}
}

func TestSession_Logout(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	err := session.Logout()
	if err != nil {
		t.Errorf("Logout() error = %v, want nil", err)
	}
}

func TestSession_MalformedEmail(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	_ = session.Mail("sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("recipient@example.com", &gosmtp.RcptOptions{})

	// Malformed email (no proper headers)
	malformed := "This is not a valid email format"
	err := session.Data(strings.NewReader(malformed))
	if err != nil {
		t.Fatalf("Data() error = %v", err)
	}

	// Should still be stored
	if s.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", s.Count())
	}

	entries := s.List()
	if entries[0].RawEmail != malformed {
		t.Errorf("RawEmail = %q, want %q", entries[0].RawEmail, malformed)
	}
}

// Authentication tests

func TestAuthConfig_HasCredentials(t *testing.T) {
	tests := []struct {
		name     string
		config   *smtp.AuthConfig
		expected bool
	}{
		{"empty", &smtp.AuthConfig{}, false},
		{"username only", &smtp.AuthConfig{Username: "user"}, false},
		{"password only", &smtp.AuthConfig{Password: "pass"}, false},
		{"both set", &smtp.AuthConfig{Username: "user", Password: "pass"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasCredentials(); got != tt.expected {
				t.Errorf("HasCredentials() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSession_AuthMechanisms_AlwaysAdvertised(t *testing.T) {
	// Auth mechanisms should always be advertised (even in open relay mode)
	// so that clients that require auth can still connect
	tests := []struct {
		name   string
		config *smtp.AuthConfig
	}{
		{"no auth (open relay)", noAuth()},
		{"with auth", withAuth("user", "pass")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New(100, 0)
			backend := smtp.NewBackend(s, tt.config)

			session, _ := backend.NewSession(nil)
			authSession := session.(gosmtp.AuthSession)

			mechs := authSession.AuthMechanisms()
			if len(mechs) != 2 {
				t.Fatalf("AuthMechanisms() returned %d mechanisms, want 2", len(mechs))
			}

			// Should support PLAIN and LOGIN
			hasPLAIN := false
			hasLOGIN := false
			for _, m := range mechs {
				if m == sasl.Plain {
					hasPLAIN = true
				}
				if m == sasl.Login {
					hasLOGIN = true
				}
			}

			if !hasPLAIN {
				t.Error("AuthMechanisms() should include PLAIN")
			}
			if !hasLOGIN {
				t.Error("AuthMechanisms() should include LOGIN")
			}
		})
	}
}

func TestSession_Auth_NoAuth_AcceptsAnything(t *testing.T) {
	// In open relay mode, any credentials should be accepted
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)
	authSession := session.(gosmtp.AuthSession)

	// Test PLAIN with random credentials
	server, err := authSession.Auth(sasl.Plain)
	if err != nil {
		t.Fatalf("Auth(PLAIN) error = %v", err)
	}

	response := []byte("\x00randomuser\x00randompass")
	_, done, err := server.Next(response)
	if err != nil {
		t.Errorf("PLAIN auth should accept any credentials in open relay mode, got error: %v", err)
	}
	if !done {
		t.Error("PLAIN auth should be done")
	}
}

func TestSession_Auth_NoAuth_LOGIN_AcceptsAnything(t *testing.T) {
	// In open relay mode, any credentials should be accepted via LOGIN
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)
	authSession := session.(gosmtp.AuthSession)

	server, err := authSession.Auth(sasl.Login)
	if err != nil {
		t.Fatalf("Auth(LOGIN) error = %v", err)
	}

	// Step through LOGIN flow with random credentials
	_, _, _ = server.Next(nil)                        // Get username prompt
	_, _, _ = server.Next([]byte("randomuser"))       // Send random username
	_, done, err := server.Next([]byte("randompass")) // Send random password

	if err != nil {
		t.Errorf("LOGIN auth should accept any credentials in open relay mode, got error: %v", err)
	}
	if !done {
		t.Error("LOGIN auth should be done")
	}
}

func TestSession_Auth_PLAIN_ValidCredentials(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, withAuth("testuser", "testpass"))

	session, _ := backend.NewSession(nil)
	authSession := session.(gosmtp.AuthSession)

	server, err := authSession.Auth(sasl.Plain)
	if err != nil {
		t.Fatalf("Auth(PLAIN) error = %v", err)
	}

	// PLAIN format: \x00username\x00password (identity is empty)
	response := []byte("\x00testuser\x00testpass")
	_, done, err := server.Next(response)

	if err != nil {
		t.Errorf("Next() error = %v, want nil", err)
	}
	if !done {
		t.Error("Next() done = false, want true")
	}
}

func TestSession_Auth_PLAIN_InvalidCredentials(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, withAuth("testuser", "testpass"))

	session, _ := backend.NewSession(nil)
	authSession := session.(gosmtp.AuthSession)

	server, _ := authSession.Auth(sasl.Plain)

	response := []byte("\x00wronguser\x00wrongpass")
	_, _, err := server.Next(response)

	if err == nil {
		t.Error("Next() should return error for invalid credentials")
	}
}

func TestSession_Auth_LOGIN_ValidCredentials(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, withAuth("testuser", "testpass"))

	session, _ := backend.NewSession(nil)
	authSession := session.(gosmtp.AuthSession)

	server, err := authSession.Auth(sasl.Login)
	if err != nil {
		t.Fatalf("Auth(LOGIN) error = %v", err)
	}

	// Step 1: Initial call, should get username prompt
	challenge, done, err := server.Next(nil)
	if err != nil {
		t.Fatalf("Step 1 error = %v", err)
	}
	if done {
		t.Error("Step 1 should not be done")
	}
	if string(challenge) != "Username:" {
		t.Errorf("Step 1 challenge = %q, want 'Username:'", string(challenge))
	}

	// Step 2: Send username, should get password prompt
	challenge, done, err = server.Next([]byte("testuser"))
	if err != nil {
		t.Fatalf("Step 2 error = %v", err)
	}
	if done {
		t.Error("Step 2 should not be done")
	}
	if string(challenge) != "Password:" {
		t.Errorf("Step 2 challenge = %q, want 'Password:'", string(challenge))
	}

	// Step 3: Send password, should complete successfully
	_, done, err = server.Next([]byte("testpass"))
	if err != nil {
		t.Errorf("Step 3 error = %v, want nil", err)
	}
	if !done {
		t.Error("Step 3 should be done")
	}
}

func TestSession_Auth_LOGIN_InvalidCredentials(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, withAuth("testuser", "testpass"))

	session, _ := backend.NewSession(nil)
	authSession := session.(gosmtp.AuthSession)

	server, _ := authSession.Auth(sasl.Login)

	// Step through the LOGIN flow with wrong credentials
	_, _, _ = server.Next(nil)                    // Get username prompt
	_, _, _ = server.Next([]byte("wronguser"))    // Send wrong username
	_, _, err := server.Next([]byte("wrongpass")) // Send wrong password

	if err == nil {
		t.Error("LOGIN should return error for invalid credentials")
	}
}

func TestSession_Auth_UnknownMechanism(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, withAuth("user", "pass"))

	session, _ := backend.NewSession(nil)
	authSession := session.(gosmtp.AuthSession)

	_, err := authSession.Auth("UNKNOWN")
	if err == nil {
		t.Error("Auth() should return error for unknown mechanism")
	}
}

func TestSession_RFC2047EncodedHeaders(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	_ = session.Mail("sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("recipient@example.com", &gosmtp.RcptOptions{})

	// Email with RFC 2047 encoded headers (Base64 UTF-8)
	// =?UTF-8?B?44Oh44O844Or44Ki44OJ44Os44K544Gu6KqN6Ki8?= decodes to "メールアドレスの認証"
	// =?UTF-8?B?QXJyb3ZlICjplovnmbrnkrDlooPvvIk=?= decodes to "Arrove (開発環境）"
	// =?UTF-8?B?55Sw5Lit6Iqx5a2Q?= decodes to "田中花子"
	emailContent := "From: =?UTF-8?B?QXJyb3ZlICjplovnmbrnkrDlooPvvIk=?= <noreply@example.com>\r\n" +
		"To: =?UTF-8?B?55Sw5Lit6Iqx5a2Q?= <recipient@example.com>\r\n" +
		"Subject: =?UTF-8?B?44Oh44O844Or44Ki44OJ44Os44K544Gu6KqN6Ki8?=\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"こんにちは"

	err := session.Data(strings.NewReader(emailContent))
	if err != nil {
		t.Fatalf("Data() error = %v", err)
	}

	entries := s.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Subject should be decoded
	expectedSubject := "メールアドレスの認証"
	if entry.Subject != expectedSubject {
		t.Errorf("Subject = %q, want %q", entry.Subject, expectedSubject)
	}

	// From header in headers map should be decoded
	fromHeaders, ok := entry.Headers["From"]
	if !ok || len(fromHeaders) == 0 {
		t.Fatal("From header not found in headers map")
	}
	expectedFrom := "Arrove (開発環境） <noreply@example.com>"
	if fromHeaders[0] != expectedFrom {
		t.Errorf("Headers[From] = %q, want %q", fromHeaders[0], expectedFrom)
	}

	// To header in headers map should be decoded
	toHeaders, ok := entry.Headers["To"]
	if !ok || len(toHeaders) == 0 {
		t.Fatal("To header not found in headers map")
	}
	expectedTo := "田中花子 <recipient@example.com>"
	if toHeaders[0] != expectedTo {
		t.Errorf("Headers[To] = %q, want %q", toHeaders[0], expectedTo)
	}
}

func TestSession_RFC2047QuotedPrintable(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	_ = session.Mail("sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("recipient@example.com", &gosmtp.RcptOptions{})

	// Email with RFC 2047 encoded Subject (Quoted-Printable UTF-8)
	// =?UTF-8?Q?=E3=83=86=E3=82=B9=E3=83=88?= decodes to "テスト"
	emailContent := "From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: =?UTF-8?Q?=E3=83=86=E3=82=B9=E3=83=88?=\r\n" +
		"\r\n" +
		"Test body"

	err := session.Data(strings.NewReader(emailContent))
	if err != nil {
		t.Fatalf("Data() error = %v", err)
	}

	entries := s.List()
	entry := entries[0]

	expectedSubject := "テスト"
	if entry.Subject != expectedSubject {
		t.Errorf("Subject = %q, want %q", entry.Subject, expectedSubject)
	}
}

func TestSession_MultipartAlternative(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	_ = session.Mail("sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("recipient@example.com", &gosmtp.RcptOptions{})

	// Multipart/alternative email with plain text and HTML parts
	emailContent := "From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: Multipart Test\r\n" +
		"Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n" +
		"\r\n" +
		"--boundary123\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"Plain text version\r\n" +
		"--boundary123\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		"<html><body><h1>HTML version</h1></body></html>\r\n" +
		"--boundary123--\r\n"

	err := session.Data(strings.NewReader(emailContent))
	if err != nil {
		t.Fatalf("Data() error = %v", err)
	}

	entries := s.List()
	entry := entries[0]

	// Should prefer HTML over plain text
	if !strings.Contains(entry.Body, "<html>") {
		t.Errorf("Body should contain HTML content, got: %q", entry.Body)
	}
	if !strings.Contains(entry.ContentType, "text/html") {
		t.Errorf("ContentType should be text/html, got: %q", entry.ContentType)
	}
}

func TestSession_MultipartBase64Encoded(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s, noAuth())

	session, _ := backend.NewSession(nil)

	_ = session.Mail("sender@example.com", &gosmtp.MailOptions{})
	_ = session.Rcpt("recipient@example.com", &gosmtp.RcptOptions{})

	// Multipart email with base64 encoded HTML part
	// "こんにちは" in base64 is "44GT44KT44Gr44Gh44Gv"
	emailContent := "From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: Base64 Test\r\n" +
		"Content-Type: multipart/alternative; boundary=\"bound\"\r\n" +
		"\r\n" +
		"--bound\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		"PGh0bWw+PGJvZHk+44GT44KT44Gr44Gh44GvPC9ib2R5PjwvaHRtbD4=\r\n" +
		"--bound--\r\n"

	err := session.Data(strings.NewReader(emailContent))
	if err != nil {
		t.Fatalf("Data() error = %v", err)
	}

	entries := s.List()
	entry := entries[0]

	// Body should be decoded from base64
	if !strings.Contains(entry.Body, "こんにちは") {
		t.Errorf("Body should contain decoded Japanese text, got: %q", entry.Body)
	}
}
