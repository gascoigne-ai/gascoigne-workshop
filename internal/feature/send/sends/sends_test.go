package sends

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func newOwner() *request.Request {
	return &request.Request{Name: "R", Protocol: "rest", Method: "GET", URL: "http://x"}
}

func noCancel() {}

func TestOnlyTheLatestSendIsAccepted(t *testing.T) {
	store := New()
	owner := newOwner()
	first, ok := store.Begin(owner, noCancel)
	if !ok || !store.Sending(owner) {
		t.Fatal("the first send starts")
	}
	if _, again := store.Begin(owner, noCancel); again {
		t.Fatal("a send while sending must not start another")
	}
	store.Cancel(owner)
	if store.Sending(owner) {
		t.Fatal("cancel ends sending")
	}
	if store.Accept(Result{Owner: owner, Number: first}) {
		t.Fatal("the late answer of a cancelled send is dropped")
	}
	second, _ := store.Begin(owner, noCancel)
	if !store.Accept(Result{Owner: owner, Number: second}) || store.Sending(owner) {
		t.Fatal("the current send is accepted and ends sending")
	}
}

func TestCancelWithoutASendChangesNothing(t *testing.T) {
	store := New()
	owner := newOwner()
	if store.Cancel(owner) {
		t.Fatal("nothing was sending")
	}
	if number, _ := store.Begin(owner, noCancel); number != 1 {
		t.Fatalf("the first send is number %d, want 1: a cancel without a send must not count", number)
	}
}

func TestNumbersAreNeverReusedAcrossOwners(t *testing.T) {
	store := New()
	seen := map[Number]bool{}
	for range 3 {
		for _, owner := range []*request.Request{newOwner(), newOwner()} {
			number, ok := store.Begin(owner, noCancel)
			if !ok || number == 0 || seen[number] {
				t.Fatalf("number %d (started %v) was already used or is zero", number, ok)
			}
			seen[number] = true
			store.Accept(Result{Owner: owner, Number: number})
		}
	}
}

// A request closed and reopened is the same owner. The answer of the send
// before the close must not end the send after it.
func TestALateResultIsDroppedWhenTheRequestIsSentAgainAfterClose(t *testing.T) {
	store := New()
	owner := newOwner()
	old, _ := store.Begin(owner, noCancel)
	store.Cancel(owner)
	current, _ := store.Begin(owner, noCancel)
	if store.Accept(Result{Owner: owner, Number: old}) {
		t.Fatal("the answer of the older send was accepted")
	}
	if !store.Sending(owner) {
		t.Fatal("the older answer ended the newer send")
	}
	if !store.Accept(Result{Owner: owner, Number: current}) {
		t.Fatal("the answer of the current send was dropped")
	}
}

func TestAResultOfAnotherOwnerIsDropped(t *testing.T) {
	store := New()
	owner, other := newOwner(), newOwner()
	number, _ := store.Begin(owner, noCancel)
	if store.Accept(Result{Owner: other, Number: number}) {
		t.Fatal("a result is accepted by its owner only")
	}
}

func TestCancelCallsTheCancelFunc(t *testing.T) {
	store := New()
	owner := newOwner()
	cancelled := 0
	store.Begin(owner, func() { cancelled++ })
	if !store.Cancel(owner) || cancelled != 1 {
		t.Fatalf("cancel reported and called %d times, want once", cancelled)
	}
	if store.Cancel(owner) || cancelled != 1 {
		t.Fatal("a second cancel has nothing to stop")
	}
}
