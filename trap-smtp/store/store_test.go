package store_test

import (
	"sync"
	"testing"
	"time"

	"github.com/probitas-test/state-servers/state-smtp/store"
)

func TestStore_Add_And_Get(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	entry := &store.EmailEntry{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test body content",
	}

	id := s.Add(entry)

	if id == "" {
		t.Error("expected non-empty ID")
	}

	got, ok := s.Get(id)
	if !ok {
		t.Fatal("expected entry to be found")
	}

	if got.From != "sender@example.com" {
		t.Errorf("From = %q, want %q", got.From, "sender@example.com")
	}
	if got.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", got.Subject, "Test Subject")
	}
	if len(got.To) != 1 || got.To[0] != "recipient@example.com" {
		t.Errorf("To = %v, want [recipient@example.com]", got.To)
	}
	if got.ReceivedAt.IsZero() {
		t.Error("expected ReceivedAt to be set")
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	_, ok := s.Get("nonexistent-id")
	if ok {
		t.Error("expected entry not to be found")
	}
}

func TestStore_List_ReturnsNewestFirst(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	entry1 := &store.EmailEntry{Subject: "First"}
	entry2 := &store.EmailEntry{Subject: "Second"}
	entry3 := &store.EmailEntry{Subject: "Third"}

	s.Add(entry1)
	s.Add(entry2)
	s.Add(entry3)

	list := s.List()

	if len(list) != 3 {
		t.Fatalf("len(list) = %d, want 3", len(list))
	}

	// Newest first
	if list[0].Subject != "Third" {
		t.Errorf("list[0].Subject = %q, want %q", list[0].Subject, "Third")
	}
	if list[1].Subject != "Second" {
		t.Errorf("list[1].Subject = %q, want %q", list[1].Subject, "Second")
	}
	if list[2].Subject != "First" {
		t.Errorf("list[2].Subject = %q, want %q", list[2].Subject, "First")
	}
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	entry := &store.EmailEntry{Subject: "Test"}
	id := s.Add(entry)

	if s.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", s.Count())
	}

	ok := s.Delete(id)
	if !ok {
		t.Error("expected Delete to return true")
	}

	if s.Count() != 0 {
		t.Errorf("Count() = %d, want 0", s.Count())
	}

	_, found := s.Get(id)
	if found {
		t.Error("expected entry to be deleted")
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	ok := s.Delete("nonexistent-id")
	if ok {
		t.Error("expected Delete to return false for nonexistent entry")
	}
}

func TestStore_Clear(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	s.Add(&store.EmailEntry{Subject: "1"})
	s.Add(&store.EmailEntry{Subject: "2"})
	s.Add(&store.EmailEntry{Subject: "3"})

	count := s.Clear()

	if count != 3 {
		t.Errorf("Clear() = %d, want 3", count)
	}
	if s.Count() != 0 {
		t.Errorf("Count() = %d, want 0", s.Count())
	}
}

func TestStore_MaxEntries_Eviction(t *testing.T) {
	t.Parallel()

	s := store.New(3, 0) // Max 3 entries

	id1 := s.Add(&store.EmailEntry{Subject: "1"})
	s.Add(&store.EmailEntry{Subject: "2"})
	s.Add(&store.EmailEntry{Subject: "3"})

	if s.Count() != 3 {
		t.Fatalf("Count() = %d, want 3", s.Count())
	}

	// Add 4th entry, should evict the oldest (id1)
	s.Add(&store.EmailEntry{Subject: "4"})

	if s.Count() != 3 {
		t.Errorf("Count() = %d, want 3", s.Count())
	}

	// First entry should be evicted
	_, found := s.Get(id1)
	if found {
		t.Error("expected oldest entry to be evicted")
	}

	// Verify order: newest first
	list := s.List()
	if list[0].Subject != "4" {
		t.Errorf("list[0].Subject = %q, want %q", list[0].Subject, "4")
	}
	if list[2].Subject != "2" {
		t.Errorf("list[2].Subject = %q, want %q", list[2].Subject, "2")
	}
}

func TestStore_Subscribe_ReceivesNewEntries(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	ch := s.Subscribe()
	defer s.Unsubscribe(ch)

	// Add entry in goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		s.Add(&store.EmailEntry{Subject: "Subscribed"})
	}()

	select {
	case entry := <-ch:
		if entry.Subject != "Subscribed" {
			t.Errorf("received entry Subject = %q, want %q", entry.Subject, "Subscribed")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for subscribed entry")
	}
}

func TestStore_Seq_MonotonicallyIncreasing(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	s.Add(&store.EmailEntry{Subject: "First"})
	s.Add(&store.EmailEntry{Subject: "Second"})
	s.Add(&store.EmailEntry{Subject: "Third"})

	list := s.List() // newest first

	if list[0].Seq != 3 {
		t.Errorf("list[0].Seq = %d, want 3", list[0].Seq)
	}
	if list[1].Seq != 2 {
		t.Errorf("list[1].Seq = %d, want 2", list[1].Seq)
	}
	if list[2].Seq != 1 {
		t.Errorf("list[2].Seq = %d, want 1", list[2].Seq)
	}
}

func TestStore_Seq_StartsAtOne(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	s.Add(&store.EmailEntry{Subject: "First"})

	entry, _ := s.Get(s.List()[0].ID)
	if entry.Seq != 1 {
		t.Errorf("first entry Seq = %d, want 1", entry.Seq)
	}
}

func TestStore_Seq_SurvivesEviction(t *testing.T) {
	t.Parallel()

	s := store.New(2, 0) // max 2 entries

	s.Add(&store.EmailEntry{Subject: "1"}) // seq=1, will be evicted
	s.Add(&store.EmailEntry{Subject: "2"}) // seq=2
	s.Add(&store.EmailEntry{Subject: "3"}) // seq=3, evicts "1"

	list := s.List() // newest first
	if list[0].Seq != 3 {
		t.Errorf("list[0].Seq = %d, want 3", list[0].Seq)
	}
	if list[1].Seq != 2 {
		t.Errorf("list[1].Seq = %d, want 2", list[1].Seq)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	s := store.New(1000, 0)

	var wg sync.WaitGroup
	numGoroutines := 100
	entriesPerGoroutine := 10

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < entriesPerGoroutine; j++ {
				s.Add(&store.EmailEntry{Subject: "Concurrent"})
			}
		}()
	}

	wg.Wait()

	expected := numGoroutines * entriesPerGoroutine
	if s.Count() != expected {
		t.Errorf("Count() = %d, want %d", s.Count(), expected)
	}
}

func TestStore_Count(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	if s.Count() != 0 {
		t.Errorf("initial Count() = %d, want 0", s.Count())
	}

	s.Add(&store.EmailEntry{})
	s.Add(&store.EmailEntry{})

	if s.Count() != 2 {
		t.Errorf("Count() = %d, want 2", s.Count())
	}
}

func TestStore_MultipleRecipients(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	entry := &store.EmailEntry{
		From: "sender@example.com",
		To:   []string{"a@example.com", "b@example.com", "c@example.com"},
	}

	id := s.Add(entry)
	got, _ := s.Get(id)

	if len(got.To) != 3 {
		t.Errorf("len(To) = %d, want 3", len(got.To))
	}
}
