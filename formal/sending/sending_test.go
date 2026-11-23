package sending_test

import (
	"context"
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
)

func conformConfig(claim formal.ClaimID) formal.ConformConfig {
	return formal.ConformConfig{Seed: formal.GivenSeed{Seed: 1}, Cases: 300, MaxSteps: 14, MinSteps: 8, Store: &memoryStore{}, Claim: claim}
}

// memoryStore keeps failing cases in memory, so no test writes a file.
type memoryStore struct{ saved []formal.Case }

func (store *memoryStore) Load(string) ([]formal.Case, error) { return store.saved, nil }
func (store *memoryStore) Save(_ string, failing formal.Case) error {
	store.saved = append(store.saved, failing)
	return nil
}

// The oracle's own properties hold in its whole space: nothing in it lets a live send have ended
// or a cancel func run twice. Conform then holds the real store to the oracle.
func TestFlightOracleIsVerified(t *testing.T) {
	formal.Check(t, flightSpec("sends"))
}

func TestSendsConformToTheOracle(t *testing.T) {
	formal.Conform(t, flightSpec("sends"), sendsBinding(quirks{}), conformConfig("one-send-per-owner"))
}

// Each broken use of the real store is a bug the store could grow; Conform must see it.
func TestBrokenUsesOfTheStoreDiverge(t *testing.T) {
	cases := []struct {
		name string
		bug  quirks
	}{
		{"begin restarts a sending owner", quirks{beginCancelsFirst: true}},
		{"accept ignores the number", quirks{acceptNewest: true}},
		{"cancel runs the func twice", quirks{cancelTwice: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := formal.RunConform("TestBroken", flightSpec("sends"), sendsBinding(tc.bug), conformConfig(""))
			if err != nil {
				t.Fatal(err)
			}
			if _, failed := outcome.(formal.ConformFailed); !failed {
				t.Fatalf("outcome = %T, want the walk to diverge from the oracle", outcome)
			}
		})
	}
}

// numberer is what a number source offers; the real store is one, and a stand-in with the rejected
// design (a counter per owner) is the other.
type numberer interface {
	Begin(owner *request.Request, cancel context.CancelFunc) (sends.Number, bool)
	Cancel(owner *request.Request) bool
	Accept(result sends.Result) bool
}

// perOwnerNumbers numbers each owner's sends from 1, the design the sends doc rejects: two owners
// then share a number.
type perOwnerNumbers struct {
	*sends.Sends
	counts map[*request.Request]sends.Number
}

func (stand perOwnerNumbers) Begin(owner *request.Request, cancel context.CancelFunc) (sends.Number, bool) {
	if _, ok := stand.Sends.Begin(owner, cancel); !ok {
		return 0, false
	}
	stand.counts[owner]++
	return stand.counts[owner], true
}

// numbersGrow runs the operations on source and reports whether every number handed out was larger
// than the one before. Operation n: kind n%3 (begin, cancel, accept the owner's latest), owner n/3.
func numbersGrow(source numberer, operations []int) bool {
	owners := [ownerCount]*request.Request{{Name: "A"}, {Name: "B"}}
	var latest [ownerCount]sends.Number
	var previous sends.Number
	for _, operation := range operations {
		owner := operation / 3
		switch operation % 3 {
		case 0:
			number, ok := source.Begin(owners[owner], func() {})
			if !ok {
				continue
			}
			if number <= previous {
				return false
			}
			previous, latest[owner] = number, number
		case 1:
			source.Cancel(owners[owner])
		case 2:
			source.Accept(sends.Result{Owner: owners[owner], Number: latest[owner]})
		}
	}
	return true
}

func operations() formal.Gen[[]int] {
	return formal.SliceOf(formal.Int(0, 3*ownerCount-1), 16)
}

func generateConfig(claim formal.ClaimID) formal.GenerateConfig {
	return formal.GenerateConfig{Seed: formal.GivenSeed{Seed: 1}, Cases: 300, Store: &memoryStore{}, Claim: claim}
}

func TestSendNumbersNeverRepeat(t *testing.T) {
	formal.ForAllWith(t, operations(), func(ops []int) bool { return numbersGrow(sends.New(), ops) }, generateConfig("send-numbers-unique"))
}

func TestPerOwnerNumbersAreFound(t *testing.T) {
	stand := func() numberer {
		return perOwnerNumbers{Sends: sends.New(), counts: map[*request.Request]sends.Number{}}
	}
	outcome, err := formal.RunForAll("TestPerOwner", operations(), func(ops []int) bool { return numbersGrow(stand(), ops) }, generateConfig(""))
	if err != nil {
		t.Fatal(err)
	}
	if _, failed := outcome.(formal.GeneratedFailed); !failed {
		t.Fatalf("outcome = %T, want a failing case for numbers that repeat across owners", outcome)
	}
}

// cancelProbe is one owner with a send in flight and a count of how often its cancel func ran.
type cancelProbe struct {
	store *sends.Sends
	owner *request.Request
	calls int
}

func newProbe(runsCancelTwice bool) func() *cancelProbe {
	return func() *cancelProbe {
		probe := &cancelProbe{store: sends.New(), owner: &request.Request{Name: "A"}}
		probe.store.Begin(probe.owner, func() {
			probe.calls++
			if runsCancelTwice {
				probe.calls++
			}
		})
		return probe
	}
}

var (
	cancelCalls = formal.Call[*cancelProbe, int]{Name: "cancel calls", Run: func(probe *cancelProbe) int { return probe.calls }}
	cancelOnce  = formal.Call[*cancelProbe, bool]{Name: "Cancel(owner)", Run: func(probe *cancelProbe) bool { return probe.store.Cancel(probe.owner) }}
)

func cancelTimes(count int) formal.Call[*cancelProbe, bool] {
	return formal.Call[*cancelProbe, bool]{Name: "Cancel(owner) repeatedly", Run: func(probe *cancelProbe) bool {
		var last bool
		for range count {
			last = probe.store.Cancel(probe.owner)
		}
		return last
	}}
}

// However often Cancel is called, the cancel func of the send runs once, and a second Cancel
// reports that nothing was sending.
func cancelBehavior() formal.Behavior[*cancelProbe] {
	return formal.Given(formal.Int(1, 4), func(count int) formal.Behavior[*cancelProbe] {
		return formal.Following(cancelCalls, cancelTimes(count)).Gives(1)
	})
}

func TestCancelRunsTheCancelFuncOnce(t *testing.T) {
	formal.HoldsWith(t, newProbe(false), cancelBehavior(), generateConfig("cancel-runs-once"))
}

func TestSecondCancelReportsNothingSending(t *testing.T) {
	again := func(bool) formal.Call[*cancelProbe, bool] { return cancelOnce }
	formal.HoldsWith(t, newProbe(false), formal.AndThen(cancelOnce, again).Gives(false), generateConfig("cancel-runs-once"))
}

func TestACancelFuncRunTwiceIsFound(t *testing.T) {
	outcome, err := formal.RunHolds("TestTwice", newProbe(true), cancelBehavior(), generateConfig(""))
	if err != nil {
		t.Fatal(err)
	}
	if _, failed := outcome.(formal.GeneratedFailed); !failed {
		t.Fatalf("outcome = %T, want a failing case", outcome)
	}
}
