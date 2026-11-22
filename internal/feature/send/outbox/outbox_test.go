package outbox

import (
	"context"
	"testing"
)

// step is one call on the outbox: an Enqueue of the named send, or a Done.
type step struct {
	enqueue  string // name of the send to enqueue; empty means Done
	wantRun  string // name of the send the call returns to start now; empty means none
	wantBusy bool   // whether the outbox has a send in flight after the call
}

func TestOutboxRunsSendsOneAtATimeInOrder(t *testing.T) {
	tests := []struct {
		name  string
		steps []step
	}{
		{"first send runs at once", []step{
			{enqueue: "a", wantRun: "a", wantBusy: true},
		}},
		{"done on an empty queue goes idle", []step{
			{enqueue: "a", wantRun: "a", wantBusy: true},
			{wantRun: "", wantBusy: false},
		}},
		{"a send behind one in flight waits", []step{
			{enqueue: "a", wantRun: "a", wantBusy: true},
			{enqueue: "b", wantRun: "", wantBusy: true},
			{wantRun: "b", wantBusy: true},
			{wantRun: "", wantBusy: false},
		}},
		{"several queued run in the order asked", []step{
			{enqueue: "a", wantRun: "a", wantBusy: true},
			{enqueue: "b", wantBusy: true},
			{enqueue: "ping", wantBusy: true},
			{enqueue: "c", wantBusy: true},
			{wantRun: "b", wantBusy: true},
			{wantRun: "ping", wantBusy: true},
			{wantRun: "c", wantBusy: true},
			{wantRun: "", wantBusy: false},
		}},
		{"after going idle the next send runs at once", []step{
			{enqueue: "a", wantRun: "a", wantBusy: true},
			{wantRun: "", wantBusy: false},
			{enqueue: "b", wantRun: "b", wantBusy: true},
		}},
		{"done on a fresh outbox stays idle", []step{
			{wantRun: "", wantBusy: false},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var outbox Outbox
			var sent []string
			for i, st := range tc.steps {
				var next Send
				var ok bool
				if st.enqueue != "" {
					name := st.enqueue
					send := Send(func(context.Context) error { sent = append(sent, name); return nil })
					next, ok = outbox.Enqueue(send)
				} else {
					next, ok = outbox.Done()
				}
				got := ""
				if ok {
					sent = nil
					_ = next(context.Background())
					got = sent[0]
				}
				if got != st.wantRun {
					t.Fatalf("step %d: started %q, want %q", i, got, st.wantRun)
				}
				if ok != (st.wantRun != "") {
					t.Fatalf("step %d: ok %v with run %q", i, ok, st.wantRun)
				}
				if outbox.Busy() != st.wantBusy {
					t.Fatalf("step %d: busy %v, want %v", i, outbox.Busy(), st.wantBusy)
				}
			}
		})
	}
}
