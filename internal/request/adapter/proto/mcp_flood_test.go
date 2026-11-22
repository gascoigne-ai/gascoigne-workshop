package proto

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// gateTransport is a transport whose sends block until release is closed or
// the transport is closed, and that counts how many sends wait at once.
type gateTransport struct {
	release  chan struct{}
	shut     chan struct{}
	shutOnce sync.Once
	inFlight atomic.Int32
	peak     atomic.Int32
	mu       sync.Mutex
	sent     []rpcMessage
}

func newGateTransport() *gateTransport {
	return &gateTransport{release: make(chan struct{}), shut: make(chan struct{})}
}

func (gate *gateTransport) send(_ context.Context, data []byte) error {
	now := gate.inFlight.Add(1)
	defer gate.inFlight.Add(-1)
	for {
		peak := gate.peak.Load()
		if now <= peak || gate.peak.CompareAndSwap(peak, now) {
			break
		}
	}
	select {
	case <-gate.release:
	case <-gate.shut:
		return fmt.Errorf("transport closed")
	}
	var message rpcMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}
	gate.mu.Lock()
	gate.sent = append(gate.sent, message)
	gate.mu.Unlock()
	return nil
}

func (gate *gateTransport) setVersion(string) {}

func (gate *gateTransport) close(context.Context) error {
	gate.shutOnce.Do(func() { close(gate.shut) })
	return nil
}

func (gate *gateTransport) messages() []rpcMessage {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	return append([]rpcMessage(nil), gate.sent...)
}

func floodSession(tr mcpTransport) *mcpSession {
	return &mcpSession{eventLog: newTestLog(1 << 16), tr: tr, waiters: map[int64]chan rpcAnswer{}}
}

func eventually(t *testing.T, what string, done func() bool) {
	t.Helper()
	for range 400 {
		if done() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// A server that sends a thousand requests at once gets at most
// maxServerWork answers in flight (plus the one goroutine that writes the
// busy replies); every request is answered, with its result or the busy
// error, and the reader (deliver) never waits for a slow send.
func TestMCPServerRequestFloodRunsBoundedAnswers(t *testing.T) {
	const requests = 1000
	gate := newGateTransport()
	session := floodSession(gate)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for id := 1; id <= requests; id++ {
			session.deliver([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"ping"}`, id)))
		}
	}()
	select {
	case <-readerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("deliver blocked on a slow send")
	}
	time.Sleep(100 * time.Millisecond) // let every goroutine that is going to start, start
	if peak := gate.peak.Load(); peak > maxServerWork+1 {
		t.Errorf("peak concurrent answers = %d, want at most %d (+1 for the busy writer)", peak, maxServerWork+1)
	}
	close(gate.release)
	eventually(t, "every request answered", func() bool { return len(gate.messages()) == requests })

	answered := map[string]bool{}
	busy := 0
	for _, reply := range gate.messages() {
		id := string(*reply.ID)
		if answered[id] {
			t.Errorf("request %s answered twice", id)
		}
		answered[id] = true
		if reply.Error != nil {
			if reply.Error.Code != rpcBusy {
				t.Errorf("request %s: error %+v, want code %d", id, reply.Error, rpcBusy)
			}
			busy++
		}
	}
	if busy == 0 {
		t.Error("no request was refused, the bound did not apply")
	}
	if notes := countNotes(session, "client busy"); notes > 2 {
		t.Errorf("%d busy notes for one flood, want it rate limited", notes)
	}
}

// countNotes counts the queued error events whose text has the prefix.
func countNotes(session *mcpSession, prefix string) int {
	count := 0
	for {
		select {
		case event := <-session.events:
			if event.Kind == request.KindError && strings.HasPrefix(event.Text, prefix) {
				count++
			}
		default:
			return count
		}
	}
}

// A thousand list_changed notifications make at most maxServerWork refresh
// requests wait at once (they are coalesced to one per list).
func TestMCPListChangedFloodBoundsRefreshes(t *testing.T) {
	gate := newGateTransport()
	session := floodSession(gate)
	for range 1000 {
		session.deliver([]byte(`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`))
	}
	time.Sleep(100 * time.Millisecond)
	if peak := gate.peak.Load(); peak > maxServerWork {
		t.Errorf("peak concurrent tools/list = %d, want at most %d", peak, maxServerWork)
	}
	close(gate.release)
	eventually(t, "a refresh", func() bool { return len(gate.messages()) > 0 })
	for _, message := range gate.messages() {
		if message.Method != "tools/list" {
			t.Errorf("sent %q, want tools/list", message.Method)
		}
	}
}

// Closing the session ends the answers blocked on the transport and frees
// every worker slot.
func TestMCPCloseReleasesWork(t *testing.T) {
	gate := newGateTransport()
	session := floodSession(gate)
	for id := 1; id <= 40; id++ {
		session.deliver([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"ping"}`, id)))
	}
	eventually(t, "answers blocked in send", func() bool { return gate.inFlight.Load() > 0 })
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	eventually(t, "work drained", func() bool { return session.work.used() == 0 && gate.inFlight.Load() == 0 })
}

// Two more sampling requests while the user has one prompt open are refused
// at once; the Sampler runs for the first only.
func TestMCPSamplingAllowsOnePromptAtATime(t *testing.T) {
	gate := newGateTransport()
	close(gate.release)
	var calls atomic.Int32
	unblock := make(chan struct{})
	session := floodSession(gate)
	session.sampler = func(ctx context.Context, call request.SamplingRequest) (request.SamplingResult, bool) {
		calls.Add(1)
		<-unblock
		return request.SamplingResult{Text: "hi"}, true
	}
	for id := 1; id <= 3; id++ {
		session.deliver([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"sampling/createMessage","params":%s}`, id, samplingParams)))
	}
	eventually(t, "two refusals", func() bool { return len(gate.messages()) == 2 })
	for _, reply := range gate.messages() {
		if reply.Error == nil || reply.Error.Code != rpcRejected || reply.Error.Message != "another sampling request is open" {
			t.Errorf("reply %+v, want the rejected error", reply)
		}
	}
	close(unblock)
	eventually(t, "the first answer", func() bool { return len(gate.messages()) == 3 })
	if got := calls.Load(); got != 1 {
		t.Errorf("Sampler ran %d times, want 1", got)
	}
}
