package smtp_test

import (
	"strings"
	"testing"

	gosmtp "github.com/emersion/go-smtp"

	"github.com/probitas-test/state-servers/state-smtp/smtp"
	"github.com/probitas-test/state-servers/state-smtp/store"
)

func TestBackend_NewSession(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s)

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
	backend := smtp.NewBackend(s)

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
	backend := smtp.NewBackend(s)

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
	backend := smtp.NewBackend(s)

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

// Note: AuthPlain is tested implicitly through the smtp.Session interface
// The Session struct implements it to accept any credentials

func TestSession_Logout(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s)

	session, _ := backend.NewSession(nil)

	err := session.Logout()
	if err != nil {
		t.Errorf("Logout() error = %v, want nil", err)
	}
}

func TestSession_MalformedEmail(t *testing.T) {
	s := store.New(100, 0)
	backend := smtp.NewBackend(s)

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
