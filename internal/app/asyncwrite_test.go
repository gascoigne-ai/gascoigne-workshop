package app

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// gateStore is a collection store whose Save waits to be released and
// keeps what it was given, so a test can see a frame does not wait for a write.
type gateStore struct {
	noStore
	release chan struct{}
	mu      sync.Mutex
	saved   []request.Collection
}

func (s *gateStore) Save(_ string, col request.Collection) error {
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
	a := testApp(t)
	gate := &gateStore{release: make(chan struct{})}
	a.store = gate
	a.setWorkspace(t.TempDir())
	now := time.Now()
	a.now = func() time.Time { return now }
	a.Step(core.Input{})
	a.cur().Req.Name = "Renamed"
	a.markStale() // the test edits the model directly
	a.Step(core.Input{})
	now = now.Add(2 * time.Second)

	done := make(chan struct{})
	go func() { a.Step(core.Input{}); close(done) }() // the write is due this frame
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the frame waited for the write")
	}
	close(gate.release)
	a.WaitWrites()
	if gate.count() != 1 || !strings.Contains(tree.Key(gate.saved[0]), "Renamed") {
		t.Fatalf("saved %d times: %+v", gate.count(), gate.saved)
	}
}

// gateHistory is a response history whose Save waits to be released.
type gateHistory struct {
	history.History
	release chan struct{}
	saves   chan string
}

func (h *gateHistory) Save(key string, _ []history.Entry) error {
	<-h.release
	h.saves <- key
	return nil
}

func TestSavingTheResponseHistoryDoesNotBlockTheFrame(t *testing.T) {
	a := testApp(t)
	gate := &gateHistory{release: make(chan struct{}), saves: make(chan string, 4)}
	a.statePath = filepath.Join(t.TempDir(), "state.json")
	a.histories = func(string) history.History { return gate }
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("{}")}

	done := make(chan struct{})
	go func() { a.saveHistory(tab); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("saving the history blocked the caller")
	}
	close(gate.release)
	a.WaitWrites()
	select {
	case <-gate.saves:
	case <-time.After(2 * time.Second):
		t.Fatal("the history was not saved")
	}
}
