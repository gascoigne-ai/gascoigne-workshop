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
