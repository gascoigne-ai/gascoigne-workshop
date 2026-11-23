package connection_test

import (
	"fmt"
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// answersState is a real Connection and the connects that are still waiting for their answer.
// Pending and Abandoned are ghosts: the answers on their way, and the connects the user gave up.
type answersState struct {
	Conn      tabs.Connection
	Pending   map[int]struct{}
	Abandoned map[int]struct{}
}

func (state answersState) clone() answersState {
	state.Pending = cloneSet(state.Pending)
	state.Abandoned = cloneSet(state.Abandoned)
	return state
}

func cloneSet(set map[int]struct{}) map[int]struct{} {
	clone := make(map[int]struct{}, len(set))
	for gen := range set {
		clone[gen] = struct{}{}
	}
	return clone
}

// answerPolicy is how the caller handles an answer that arrives. The window drops an answer of a
// connect that is no longer current; a broken policy swallows a current failure.
type answerPolicy struct {
	swallowsFailure bool
}

// answers is the actions around a connect: start, give up, an answer arriving (opened or failed,
// each a choice), the session ending. The caller's one rule is the filter: an answer of a connect
// that is not Current is dropped before the connection sees it.
func answerActions(impl ops, policy answerPolicy) []formal.Action[answersState] {
	actions := []formal.Action[answersState]{
		{Name: "Begin", Next: func(state answersState) []answersState {
			state = state.clone()
			gen, ok := impl.begin(&state.Conn)
			if overGen(state.Conn) {
				return nil
			}
			if ok {
				state.Pending[gen] = struct{}{}
			}
			return []answersState{state}
		}},
		{Name: "Disconnect", Next: func(state answersState) []answersState {
			state = state.clone()
			before := state.Conn.Gen
			_, abandoned := impl.disconnect(&state.Conn)
			if abandoned {
				delete(state.Pending, before)
				state.Abandoned[before] = struct{}{}
			}
			return []answersState{state}
		}},
		{Name: "SessionEnds", Next: func(state answersState) []answersState {
			if state.Conn.Session() == nil {
				return nil
			}
			state.Conn.Closed()
			return []answersState{state.clone()}
		}},
	}
	for gen := 1; gen <= connects; gen++ {
		actions = append(actions, formal.Action[answersState]{
			Name:     fmt.Sprintf("Answer(%d)", gen),
			Fairness: formal.WeakFair,
			Next: func(state answersState) []answersState {
				if _, waiting := state.Pending[gen]; !waiting {
					return nil
				}
				state = state.clone()
				delete(state.Pending, gen)
				if !state.Conn.Current(gen) {
					return []answersState{state} // dropped: the connection never sees it
				}
				opened, failed := state.clone(), state.clone()
				impl.opened(&opened.Conn, stubSession{Gen: gen})
				if policy.swallowsFailure {
					return []answersState{opened, failed}
				}
				impl.failed(&failed.Conn)
				return []answersState{opened, failed}
			},
		})
	}
	return actions
}

func answersSpec(name string, impl ops, policy answerPolicy) formal.Spec[answersState] {
	var init []answersState
	for _, conn := range both() {
		init = append(init, answersState{Conn: conn, Pending: map[int]struct{}{}, Abandoned: map[int]struct{}{}})
	}
	return formal.Spec[answersState]{
		Name:    name,
		Init:    init,
		Actions: answerActions(impl, policy),
		Always: []formal.Property[answersState]{
			{Name: "AbandonedNeverCurrent", Claim: "stale-connect-not-current", Holds: func(state answersState) bool {
				for gen := range state.Abandoned {
					if state.Conn.Current(gen) {
						return false
					}
				}
				return true
			}},
			{Name: "OpenSessionIsLatest", Claim: "stale-connect-not-current", Holds: func(state answersState) bool {
				session, ok := state.Conn.Session().(stubSession)
				return state.Conn.Session() == nil || (ok && session.Gen == state.Conn.Gen)
			}},
			{Name: "ConnectingHasAnAnswerComing", Claim: "connecting-leaves", Holds: func(state answersState) bool {
				if state.Conn.State() != tabs.ConnConnecting {
					return true
				}
				_, waiting := state.Pending[state.Conn.Gen]
				return waiting
			}},
		},
		Done:  func(answersState) bool { return true },
		Space: formal.Space{Params: []formal.Param{{Name: "connects", Value: connects}}},
	}
}

// The caller filters an answer by Current, so what must hold of the real Connection is that a
// connect given up is never current again, that an open session is the latest connect's, and that a
// connect in progress always has its answer on the way.
func TestConnectAnswersAreVerified(t *testing.T) {
	formal.Check(t, answersSpec("connection-answers", realOps(), answerPolicy{}))
}

// Each broken variant is a bug that makes a stale answer current or loses an answer.
func TestBrokenAnswerHandlingIsFound(t *testing.T) {
	disconnectKeepsGen := realOps()
	disconnectKeepsGen.disconnect = func(conn *tabs.Connection) (request.Session, bool) {
		gen := conn.Gen
		session, abandoned := conn.Disconnect()
		conn.Gen = gen
		return session, abandoned
	}
	cases := []struct {
		name         string
		impl         ops
		policy       answerPolicy
		wantProperty string
	}{
		{"Disconnect of a connect keeps its number", disconnectKeepsGen, answerPolicy{}, "AbandonedNeverCurrent"},
		{"a current failure is swallowed", realOps(), answerPolicy{swallowsFailure: true}, "ConnectingHasAnAnswerComing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := formal.Run(answersSpec("connection-answers-broken", tc.impl, tc.policy))
			if err != nil {
				t.Fatal(err)
			}
			violated, ok := outcome.(formal.Violated)
			if !ok {
				t.Fatalf("outcome = %T, want a violation", outcome)
			}
			broken, ok := violated.Violation.(formal.InvariantBroken)
			if !ok || broken.Property != tc.wantProperty {
				t.Fatalf("violation = %#v, want property %q broken", violated.Violation, tc.wantProperty)
			}
		})
	}
}
