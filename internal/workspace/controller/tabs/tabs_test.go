package tabs

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func newRequest() *model.Request {
	return &model.Request{Name: "R", Protocol: "rest", Method: "GET", URL: "http://x"}
}

func TestNewTabInfersABodyTypeForOldRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"no body", "", model.BodyNone},
		{"a body from before body types is text", "hello", model.BodyText},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := newRequest()
			req.Body = test.body
			if tab := New(req); tab.Req.BodyType != test.want {
				t.Fatalf("body type = %q, want %q", tab.Req.BodyType, test.want)
			}
		})
	}
}

func TestDirtyFollowsEditsAndMarkSaved(t *testing.T) {
	tab := New(newRequest())
	if tab.Dirty() {
		t.Fatal("a new tab is clean")
	}
	tab.Req.URL = "http://y"
	if !tab.Dirty() {
		t.Fatal("an edit makes the tab dirty")
	}
	tab.MarkSaved()
	if tab.Dirty() {
		t.Fatal("saving makes the tab clean")
	}
}

func TestOnlyTheLatestSendIsAccepted(t *testing.T) {
	tab := New(newRequest())
	first, ok := tab.BeginSend()
	if !ok || !tab.Sending {
		t.Fatal("the first send starts")
	}
	if _, again := tab.BeginSend(); again {
		t.Fatal("a send while sending must not start another")
	}
	tab.Cancel()
	if tab.Sending || tab.Resp == nil || tab.Resp.Err != model.ErrCancelled {
		t.Fatalf("cancel: sending %v resp %+v", tab.Sending, tab.Resp)
	}
	if tab.Accept(first) {
		t.Fatal("the late answer of a cancelled send is dropped")
	}
	second, _ := tab.BeginSend()
	if !tab.Accept(second) || tab.Sending {
		t.Fatal("the current send is accepted and ends sending")
	}
}

func TestCancelWithoutASendChangesNothing(t *testing.T) {
	tab := New(newRequest())
	tab.Cancel()
	if tab.Resp != nil || tab.Seq != 0 {
		t.Fatalf("resp %+v seq %d", tab.Resp, tab.Seq)
	}
}

func TestNextWrapsAtBothEnds(t *testing.T) {
	tests := []struct {
		name                string
		active, count, step int
		want                int
	}{
		{"forward", 0, 3, 1, 1},
		{"forward wraps", 2, 3, 1, 0},
		{"back", 2, 3, -1, 1},
		{"back wraps", 0, 3, -1, 2},
		{"one tab stays", 0, 1, 1, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Next(test.active, test.count, test.step); got != test.want {
				t.Fatalf("Next(%d, %d, %d) = %d, want %d", test.active, test.count, test.step, got, test.want)
			}
		})
	}
}

func TestAfterCloseKeepsTheSameTabActiveWhenPossible(t *testing.T) {
	tests := []struct {
		name                      string
		active, closed, remaining int
		want                      int
	}{
		{"closing the last leaves the first slot", 0, 0, 0, 0},
		{"closing a tab before the active one shifts it", 2, 0, 2, 1},
		{"closing the active one picks the next", 1, 1, 2, 1},
		{"closing the active last one picks the new last", 2, 2, 2, 1},
		{"closing a tab after the active one changes nothing", 0, 2, 2, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AfterClose(test.active, test.closed, test.remaining); got != test.want {
				t.Fatalf("AfterClose(%d, %d, %d) = %d, want %d", test.active, test.closed, test.remaining, got, test.want)
			}
		})
	}
}

func TestAfterMoveFollowsTheActiveTab(t *testing.T) {
	tests := []struct {
		name                   string
		active, from, to, want int
	}{
		{"the moved tab is active", 0, 0, 2, 2},
		{"a tab moves from before to after the active one", 1, 0, 2, 0},
		{"a tab moves from after to before the active one", 1, 2, 0, 2},
		{"a move that does not cross it changes nothing", 0, 1, 2, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AfterMove(test.active, test.from, test.to); got != test.want {
				t.Fatalf("AfterMove(%d, %d, %d) = %d, want %d", test.active, test.from, test.to, got, test.want)
			}
		})
	}
}
