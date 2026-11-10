package ui

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// gateStore is a collection store whose Save waits to be released and
// keeps what it was given, so a test can see a frame does not wait for a write.
type gateStore struct {
	noStore
	release chan struct{}
	mu      sync.Mutex
	saved   []model.Collection
}

func (s *gateStore) Save(_ string, col model.Collection) error {
	<-s.release
	s.mu.Lock()
	s.saved = append(s.saved, col)
	s.mu.Unlock()
	return nil
}

func (s *gateStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.saved)
}

func TestTheAutosaveOfTheCollectionDoesNotBlockTheFrame(t *testing.T) {
	a, _ := newTestApp(t)
	gate := &gateStore{release: make(chan struct{})}
	a.store, a.workspace = gate, t.TempDir()
	now := time.Now()
	a.now = func() time.Time { return now }
	a.step(core.Input{})
	a.cur().Req.Name = "Renamed"
	a.markStale() // the test edits the model directly
	a.step(core.Input{})
	now = now.Add(2 * time.Second)

	done := make(chan struct{})
	go func() { a.step(core.Input{}); close(done) }() // the write is due this frame
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the frame waited for the write")
	}
	close(gate.release)
	a.waitWrites()
	if gate.count() != 1 || !strings.Contains(tree.Key(gate.saved[0]), "Renamed") {
		t.Fatalf("saved %d times: %+v", gate.count(), gate.saved)
	}
}
