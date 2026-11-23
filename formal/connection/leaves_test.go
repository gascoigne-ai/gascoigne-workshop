package connection_test

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// leavesSpec starts right after the first connect was asked for and says the connection leaves
// Connecting on every infinite behavior, provided answers that arrive are taken (weakly fair).
// The library has no leads-to, so the claim covers the first connect; a later connect is the safety
// property ConnectingHasAnAnswerComing of the answers spec.
func leavesSpec(name string, policy answerPolicy) formal.Spec[answersState] {
	spec := answersSpec(name, realOps(), policy)
	spec.Always = nil
	spec.Init = nil
	for _, conn := range both() {
		started := answersState{Conn: conn, Pending: map[int]struct{}{}, Abandoned: map[int]struct{}{}}
		gen, _ := started.Conn.Begin()
		started.Pending[gen] = struct{}{}
		spec.Init = append(spec.Init, started)
	}
	spec.Eventually = []formal.Property[answersState]{
		{Name: "LeavesConnecting", Claim: "connecting-leaves", Holds: func(state answersState) bool {
			return state.Conn.State() != tabs.ConnConnecting
		}},
	}
	return spec
}

func TestConnectingAlwaysLeaves(t *testing.T) {
	formal.Check(t, leavesSpec("connecting-leaves", answerPolicy{}))
}

// An answer that nobody takes in leaves the connection in Connecting for ever; the check must find
// that as a lasso.
func TestASwallowedAnswerLeavesItConnectingForEver(t *testing.T) {
	outcome, err := formal.Run(leavesSpec("connecting-stuck", answerPolicy{swallowsFailure: true}))
	if err != nil {
		t.Fatal(err)
	}
	violated, ok := outcome.(formal.Violated)
	if !ok {
		t.Fatalf("outcome = %T, want a violation", outcome)
	}
	if _, ok := violated.Violation.(formal.LivenessBroken); !ok {
		t.Fatalf("violation = %#v, want liveness broken", violated.Violation)
	}
}
