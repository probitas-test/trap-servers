package store

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Attachment represents an email attachment
type Attachment struct {
	ID          string `json:"id"`           // Unique ID for download
	Filename    string `json:"filename"`     // Original filename
	ContentType string `json:"content_type"` // MIME type
	Size        int    `json:"size"`         // Size in bytes
	ContentID   string `json:"content_id"`   // Content-ID for inline images (cid:xxx)
	IsInline    bool   `json:"is_inline"`    // True if inline attachment
	Data        []byte `json:"-"`            // Binary data (not serialized to JSON)
}

// EmailEntry represents a single received email
type EmailEntry struct {
	ID         string    `json:"id"`
	ReceivedAt time.Time `json:"received_at"`

	// SMTP envelope
	From string   `json:"from"`
	To   []string `json:"to"`

	// Parsed headers
	Subject     string              `json:"subject"`
	Date        string              `json:"date"`
	MessageID   string              `json:"message_id"`
	ContentType string              `json:"content_type"`
	Headers     map[string][]string `json:"headers"`

	// Body content (parsed from MIME)
	Body     string `json:"body"`      // Primary body (HTML if available, otherwise Text)
	HtmlBody string `json:"html_body"` // HTML version (empty if not available)
	TextBody string `json:"text_body"` // Plain text version (empty if not available)
	RawEmail string `json:"raw_email"`

	// Attachments
	Attachments []Attachment `json:"attachments"`
}

// Store manages email entries in memory
type Store struct {
	mu         sync.RWMutex
	entries    map[string]*EmailEntry
	order      []string // maintains insertion order for FIFO eviction
	maxEntries int
	ttl        time.Duration
	listeners  map[chan *EmailEntry]struct{}
}

// New creates a new Store instance
func New(maxEntries int, ttlSeconds int) *Store {
	s := &Store{
		entries:    make(map[string]*EmailEntry),
		order:      make([]string, 0),
		maxEntries: maxEntries,
		ttl:        time.Duration(ttlSeconds) * time.Second,
		listeners:  make(map[chan *EmailEntry]struct{}),
	}

	// Start cleanup goroutine if TTL is set
	if ttlSeconds > 0 {
		go s.cleanupLoop()
	}

	return s
}

// Add stores a new email entry and returns its ID
func (s *Store) Add(entry *EmailEntry) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.ReceivedAt.IsZero() {
		entry.ReceivedAt = time.Now()
	}

	// Evict oldest entries if at capacity
	for s.maxEntries > 0 && len(s.order) >= s.maxEntries {
		oldestID := s.order[0]
		s.order = s.order[1:]
		delete(s.entries, oldestID)
	}

	s.entries[entry.ID] = entry
	s.order = append(s.order, entry.ID)

	// Notify listeners
	for ch := range s.listeners {
		select {
		case ch <- entry:
		default:
			// Skip if channel is full
		}
	}

	return entry.ID
}

// Get retrieves a specific entry by ID
func (s *Store) Get(id string) (*EmailEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[id]
	return entry, ok
}

// List returns all entries in reverse chronological order (newest first)
func (s *Store) List() []*EmailEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*EmailEntry, len(s.order))
	for i, id := range s.order {
		result[len(s.order)-1-i] = s.entries[id]
	}
	return result
}

// Delete removes an entry by ID
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[id]; !ok {
		return false
	}

	delete(s.entries, id)

	// Remove from order slice
	for i, oid := range s.order {
		if oid == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}

	return true
}

// Clear removes all entries
func (s *Store) Clear() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := len(s.entries)
	s.entries = make(map[string]*EmailEntry)
	s.order = make([]string, 0)
	return count
}

// Count returns the number of entries
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// Subscribe returns a channel that receives new entries
func (s *Store) Subscribe() chan *EmailEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *EmailEntry, 10)
	s.listeners[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a listener channel
func (s *Store) Unsubscribe(ch chan *EmailEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.listeners, ch)
	close(ch)
}

// cleanupLoop periodically removes expired entries
func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ttl == 0 {
		return
	}

	cutoff := time.Now().Add(-s.ttl)
	newOrder := make([]string, 0, len(s.order))

	for _, id := range s.order {
		entry := s.entries[id]
		if entry.ReceivedAt.After(cutoff) {
			newOrder = append(newOrder, id)
		} else {
			delete(s.entries, id)
		}
	}

	s.order = newOrder
}
