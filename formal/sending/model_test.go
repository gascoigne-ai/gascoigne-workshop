package sending_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
)

// TestMain draws nothing, so a failing run prints plain text on every terminal.
func TestMain(m *testing.M) {
	os.Setenv(formal.VisualEnv, "off")
	os.Exit(m.Run())
}

const (
	ownerCount = 2
	sendCount  = 3 // model send ids run 1..sendCount; 0 is "none"
)

// End says how a send of the model ended; the zero value is "has not ended".
type End uint8

const (
	NotEnded End = iota
	Cancelled
	Accepted
)

// seen is what the real store shows through its API, so Conform compares it to the model.
type seen struct {
	Sending     [ownerCount]bool
	Answer      bool // what the last Begin, Cancel or Accept returned
	CancelCalls [sendCount + 1]int
}

// ghost is what the model knows and the store does not show: the live send of each owner by model
// id, the results still on their way to the bus, and how each send ended. Properties read it.
type ghost struct {
	Live    [ownerCount]int
	Started int
	OnBus   map[delivery]struct{}
	Ended   [sendCount + 1]End
}

// delivery is the result of send Send of owner Owner, on its way to the bus.
type delivery struct{ Owner, Send int }

// flightState is the model state: the part the code shows and the ghost beside it.
type flightState struct {
	Seen  seen
	Ghost ghost
}

func (state flightState) clone() flightState {
	bus := make(map[delivery]struct{}, len(state.Ghost.OnBus))
	for key := range state.Ghost.OnBus {
		bus[key] = struct{}{}
	}
	state.Ghost.OnBus = bus
	return state
}

func (state flightState) setSending() flightState {
	for owner, live := range state.Ghost.Live {
		state.Seen.Sending[owner] = live != 0
	}
	return state
}

func begin(owner int) formal.Action[flightState] {
	return formal.Action[flightState]{Name: fmt.Sprintf("Begin(%d)", owner), Next: func(state flightState) []flightState {
		state = state.clone()
		if state.Ghost.Live[owner] != 0 {
			state.Seen.Answer = false
			return []flightState{state}
		}
		if state.Ghost.Started == sendCount {
			return nil
		}
		state.Ghost.Started++
		id := state.Ghost.Started
		state.Ghost.Live[owner] = id
		state.Ghost.OnBus[delivery{owner, id}] = struct{}{}
		state.Seen.Answer = true
		return []flightState{state.setSending()}
	}}
}

func cancel(owner int) formal.Action[flightState] {
	return formal.Action[flightState]{Name: fmt.Sprintf("Cancel(%d)", owner), Next: func(state flightState) []flightState {
		state = state.clone()
		id := state.Ghost.Live[owner]
		state.Seen.Answer = id != 0
		if id != 0 {
			state.Seen.CancelCalls[id]++
			state.Ghost.Ended[id] = Cancelled
			state.Ghost.Live[owner] = 0
		}
		return []flightState{state.setSending()}
	}}
}

// deliver is the arrival of a result: it is wanted only when it is of the send that is live.
func deliver(owner, id int) formal.Action[flightState] {
	return formal.Action[flightState]{Name: fmt.Sprintf("Deliver(%d,%d)", owner, id), Next: func(state flightState) []flightState {
		key := delivery{owner, id}
		if _, onBus := state.Ghost.OnBus[key]; !onBus {
			return nil
		}
		state = state.clone()
		delete(state.Ghost.OnBus, key)
		state.Seen.Answer = state.Ghost.Live[owner] == id
		if state.Seen.Answer {
			state.Ghost.Ended[id] = Accepted
			state.Ghost.Live[owner] = 0
		}
		return []flightState{state.setSending()}
	}}
}

// flightSpec is the oracle: owners that send one at a time, cancel, and receive results that may
// be late. It is small on purpose: the claims about the real store are checked by Conform, which
// compares the store to this after every step.
func flightSpec(name string) formal.Spec[flightState] {
	var actions []formal.Action[flightState]
	for owner := range ownerCount {
		actions = append(actions, begin(owner), cancel(owner))
		for id := 1; id <= sendCount; id++ {
			actions = append(actions, deliver(owner, id))
		}
	}
	return formal.Spec[flightState]{
		Name:    name,
		Init:    []flightState{{Ghost: ghost{OnBus: map[delivery]struct{}{}}}},
		Actions: actions,
		Always: []formal.Property[flightState]{
			{Name: "LiveSendHasNotEnded", Holds: func(state flightState) bool {
				for _, id := range state.Ghost.Live {
					if id != 0 && state.Ghost.Ended[id] != NotEnded {
						return false
					}
				}
				return true
			}},
			{Name: "CancelCalledOnce", Holds: func(state flightState) bool {
				for _, calls := range state.Seen.CancelCalls {
					if calls > 1 {
						return false
					}
				}
				return true
			}},
		},
		Done: func(flightState) bool { return true },
		Space: formal.Space{Params: []formal.Param{
			{Name: "owners", Value: ownerCount}, {Name: "sends", Value: sendCount},
		}},
	}
}

// quirks are the bugs a broken binding plants into the real store's use; all false is the real use.
type quirks struct {
	beginCancelsFirst bool // a Begin of a sending owner cancels the old send and starts again
	acceptNewest      bool // a result is accepted under the owner's newest number, not its own
	cancelTwice       bool // the cancel func is run a second time by the caller
}

// storeUnderTest is the real store with the owners and numbers the model's ids map to.
type storeUnderTest struct {
	store       *sends.Sends
	owners      [ownerCount]*request.Request
	numbers     [sendCount + 1]sends.Number // real number of each model send id
	newest      [ownerCount]sends.Number    // the number of the latest send of each owner
	answer      bool
	cancelCalls [sendCount + 1]int
	ghost       ghost
}

func newStoreUnderTest() *storeUnderTest {
	under := &storeUnderTest{store: sends.New(), ghost: ghost{OnBus: map[delivery]struct{}{}}}
	for owner := range under.owners {
		under.owners[owner] = &request.Request{Name: fmt.Sprintf("R%d", owner), Protocol: "rest", Method: "GET", URL: "http://x"}
	}
	return under
}

func (under *storeUnderTest) observe() flightState {
	state := flightState{Ghost: under.ghost}
	state.Seen.Answer = under.answer
	state.Seen.CancelCalls = under.cancelCalls
	for owner := range under.owners {
		state.Seen.Sending[owner] = under.store.Sending(under.owners[owner])
	}
	return state
}

func sendsBinding(bug quirks) formal.Binding[flightState, *storeUnderTest] {
	var steps []formal.BoundStep[flightState, *storeUnderTest]
	for owner := range ownerCount {
		steps = append(steps,
			formal.BoundStep[flightState, *storeUnderTest]{
				Action: fmt.Sprintf("Begin(%d)", owner),
				Apply: func(under *storeUnderTest, before, after flightState) {
					id := before.Ghost.Started + 1
					startCancel := func() { under.cancelCalls[id]++ }
					if bug.beginCancelsFirst {
						under.store.Cancel(under.owners[owner])
					}
					number, ok := under.store.Begin(under.owners[owner], startCancel)
					under.answer = ok
					if ok {
						under.numbers[id], under.newest[owner] = number, number
					}
					under.ghost = after.Ghost
				},
			},
			formal.BoundStep[flightState, *storeUnderTest]{
				Action: fmt.Sprintf("Cancel(%d)", owner),
				Apply: func(under *storeUnderTest, _, after flightState) {
					under.answer = under.store.Cancel(under.owners[owner])
					if bug.cancelTwice && under.answer {
						under.cancelCalls[under.lastBegun(owner)]++
					}
					under.ghost = after.Ghost
				},
			},
		)
		for id := 1; id <= sendCount; id++ {
			steps = append(steps, formal.BoundStep[flightState, *storeUnderTest]{
				Action: fmt.Sprintf("Deliver(%d,%d)", owner, id),
				Apply: func(under *storeUnderTest, _, after flightState) {
					number := under.numbers[id]
					if bug.acceptNewest {
						number = under.newest[owner]
					}
					under.answer = under.store.Accept(sends.Result{Owner: under.owners[owner], Number: number})
					under.ghost = after.Ghost
				},
			})
		}
	}
	return formal.Binding[flightState, *storeUnderTest]{
		New:     newStoreUnderTest,
		Observe: (*storeUnderTest).observe,
		Steps:   steps,
	}
}

// lastBegun is the model id of the latest send that started for owner.
func (under *storeUnderTest) lastBegun(owner int) int {
	for id := sendCount; id >= 1; id-- {
		if under.numbers[id] == under.newest[owner] && under.numbers[id] != 0 {
			return id
		}
	}
	return 0
}
