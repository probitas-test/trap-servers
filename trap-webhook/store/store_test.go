package store_test

import (
	"sync"
	"testing"
	"time"

	"github.com/probitas-test/state-servers/trap-webhook/store"
)

func TestStore_Add_And_Get(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	entry := &store.WebhookEntry{
		Method: "POST",
		Path:   "/webhook/test",
		Body:   `{"message": "hello"}`,
	}

	id := s.Add(entry)

	if id == "" {
		t.Error("expected non-empty ID")
	}

	got, ok := s.Get(id)
	if !ok {
		t.Fatal("expected entry to be found")
	}

	if got.Method != "POST" {
		t.Errorf("Method = %q, want %q", got.Method, "POST")
	}
	if got.Path != "/webhook/test" {
		t.Errorf("Path = %q, want %q", got.Path, "/webhook/test")
	}
	if got.Body != `{"message": "hello"}` {
		t.Errorf("Body = %q, want %q", got.Body, `{"message": "hello"}`)
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

	// Add entries with small delay to ensure different timestamps
	entry1 := &store.WebhookEntry{Path: "/first"}
	entry2 := &store.WebhookEntry{Path: "/second"}
	entry3 := &store.WebhookEntry{Path: "/third"}

	s.Add(entry1)
	s.Add(entry2)
	s.Add(entry3)

	list := s.List()

	if len(list) != 3 {
		t.Fatalf("len(list) = %d, want 3", len(list))
	}

	// Newest first
	if list[0].Path != "/third" {
		t.Errorf("list[0].Path = %q, want %q", list[0].Path, "/third")
	}
	if list[1].Path != "/second" {
		t.Errorf("list[1].Path = %q, want %q", list[1].Path, "/second")
	}
	if list[2].Path != "/first" {
		t.Errorf("list[2].Path = %q, want %q", list[2].Path, "/first")
	}
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	entry := &store.WebhookEntry{Path: "/test"}
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

	s.Add(&store.WebhookEntry{Path: "/1"})
	s.Add(&store.WebhookEntry{Path: "/2"})
	s.Add(&store.WebhookEntry{Path: "/3"})

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

	id1 := s.Add(&store.WebhookEntry{Path: "/1"})
	s.Add(&store.WebhookEntry{Path: "/2"})
	s.Add(&store.WebhookEntry{Path: "/3"})

	if s.Count() != 3 {
		t.Fatalf("Count() = %d, want 3", s.Count())
	}

	// Add 4th entry, should evict the oldest (id1)
	s.Add(&store.WebhookEntry{Path: "/4"})

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
	if list[0].Path != "/4" {
		t.Errorf("list[0].Path = %q, want %q", list[0].Path, "/4")
	}
	if list[2].Path != "/2" {
		t.Errorf("list[2].Path = %q, want %q", list[2].Path, "/2")
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
		s.Add(&store.WebhookEntry{Path: "/subscribed"})
	}()

	select {
	case entry := <-ch:
		if entry.Path != "/subscribed" {
			t.Errorf("received entry Path = %q, want %q", entry.Path, "/subscribed")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for subscribed entry")
	}
}

func TestStore_Seq_MonotonicallyIncreasing(t *testing.T) {
	t.Parallel()

	s := store.New(100, 0)

	s.Add(&store.WebhookEntry{Path: "/first"})
	s.Add(&store.WebhookEntry{Path: "/second"})
	s.Add(&store.WebhookEntry{Path: "/third"})

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

	s.Add(&store.WebhookEntry{Path: "/first"})

	entry, _ := s.Get(s.List()[0].ID)
	if entry.Seq != 1 {
		t.Errorf("first entry Seq = %d, want 1", entry.Seq)
	}
}

func TestStore_Seq_SurvivesEviction(t *testing.T) {
	t.Parallel()

	s := store.New(2, 0) // max 2 entries

	s.Add(&store.WebhookEntry{Path: "/1"}) // seq=1, will be evicted
	s.Add(&store.WebhookEntry{Path: "/2"}) // seq=2
	s.Add(&store.WebhookEntry{Path: "/3"}) // seq=3, evicts "/1"

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
				s.Add(&store.WebhookEntry{Path: "/concurrent"})
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

	s.Add(&store.WebhookEntry{})
	s.Add(&store.WebhookEntry{})

	if s.Count() != 2 {
		t.Errorf("Count() = %d, want 2", s.Count())
	}
}
