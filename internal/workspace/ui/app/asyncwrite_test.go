package ui

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
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

// gateHistory is a response history whose Save waits to be released.
type gateHistory struct {
	rport.History
	release chan struct{}
	saves   chan string
}

func (h *gateHistory) Save(key string, _ []rport.Entry) error {
	<-h.release
	h.saves <- key
	return nil
}

func TestSavingTheResponseHistoryDoesNotBlockTheFrame(t *testing.T) {
	a, _ := newTestApp(t)
	gate := &gateHistory{release: make(chan struct{}), saves: make(chan string, 4)}
	a.statePath = filepath.Join(t.TempDir(), "state.json")
	a.histories = func(string) rport.History { return gate }
	tab := a.cur()
	tab.Resp = &model.Response{Status: 200, Body: []byte("{}")}

	done := make(chan struct{})
	go func() { a.saveHistory(tab); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("saving the history blocked the caller")
	}
	close(gate.release)
	a.waitWrites()
	select {
	case <-gate.saves:
	case <-time.After(2 * time.Second):
		t.Fatal("the history was not saved")
	}
}
