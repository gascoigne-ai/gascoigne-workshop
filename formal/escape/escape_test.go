package escape_test

import (
	"os"
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/overlay"
)

// TestMain draws nothing, so a failing run prints plain text on every terminal.
func TestMain(m *testing.M) {
	os.Setenv(formal.VisualEnv, "off")
	os.Exit(m.Run())
}

// layerCount is how many layers Escape can close; the model is built from it, so a new layer
// widens the checked space instead of escaping it.
var layerCount = len(overlay.Open{})

// holdsKeyboardLayers is the written list of layers whose keys the text under them must not act
// on. It is kept apart from the code on purpose: a reordered constant changes the code, not this.
var holdsKeyboardLayers = []overlay.Layer{
	overlay.ContextMenu, overlay.Dialog, overlay.DestPicker, overlay.FileBrowser, overlay.CallLog,
}

// escapeState is a set of open layers. Pressed and Started are ghosts: how many layers Escape has
// closed and how many were open at the start, read by properties only.
type escapeState struct {
	Open    overlay.Open
	Pressed int
	Started int
}

// overlayImpl is the part of overlay the model calls; the real one is the two methods, a broken
// variant replaces one.
type overlayImpl struct {
	top   func(overlay.Open) (overlay.Layer, bool)
	holds func(overlay.Open) bool
}

func realOverlay() overlayImpl {
	return overlayImpl{top: overlay.Open.Top, holds: overlay.Open.HoldsKeyboard}
}

func openCount(open overlay.Open) int {
	count := 0
	for _, isOpen := range open {
		if isOpen {
			count++
		}
	}
	return count
}

// everySet is every combination of open layers: Escape must empty each one, so each is a start.
func everySet() []escapeState {
	var states []escapeState
	for bits := 0; bits < 1<<layerCount; bits++ {
		var open overlay.Open
		for layer := range open {
			open[layer] = bits&(1<<layer) != 0
		}
		states = append(states, escapeState{Open: open, Started: openCount(open)})
	}
	return states
}

func holdsKeyboardByList(open overlay.Open) bool {
	for _, layer := range holdsKeyboardLayers {
		if open[layer] {
			return true
		}
	}
	return false
}

func escapeSpec(name string, impl overlayImpl) formal.Spec[escapeState] {
	return formal.Spec[escapeState]{
		Name: name,
		Init: everySet(),
		Actions: []formal.Action[escapeState]{{Name: "Escape", Next: func(state escapeState) []escapeState {
			top, anyOpen := impl.top(state.Open)
			if !anyOpen {
				return nil
			}
			state.Open[top] = false
			state.Pressed++
			return []escapeState{state}
		}}},
		Always: []formal.Property[escapeState]{
			{Name: "TopIsFirstOpen", Claim: "escape-closes-topmost", Holds: func(state escapeState) bool {
				top, anyOpen := impl.top(state.Open)
				if anyOpen != (openCount(state.Open) > 0) {
					return false
				}
				if !anyOpen {
					return true
				}
				for layer := overlay.Layer(0); layer < top; layer++ {
					if state.Open[layer] {
						return false
					}
				}
				return state.Open[top]
			}},
			{Name: "EachEscapeClosesOne", Claim: "escape-never-stuck", Holds: func(state escapeState) bool {
				return openCount(state.Open)+state.Pressed == state.Started
			}},
			{Name: "KeyboardHeldByListedLayers", Claim: "keyboard-held-by-overlays", Holds: func(state escapeState) bool {
				return impl.holds(state.Open) == holdsKeyboardByList(state.Open)
			}},
			{Name: "KeyboardFollowsTop", Claim: "keyboard-held-by-overlays", Holds: func(state escapeState) bool {
				top, anyOpen := impl.top(state.Open)
				return impl.holds(state.Open) == (anyOpen && top <= overlay.CallLog)
			}},
		},
		Done:  func(state escapeState) bool { return openCount(state.Open) == 0 },
		Space: formal.Space{Params: []formal.Param{{Name: "layers", Value: layerCount}}},
	}
}

// Every set of open layers is a start and the real Top and HoldsKeyboard are the model's
// transition and properties, so a pass is a proof about the real methods for all 2^layers sets.
func TestOverlayEscapeIsVerified(t *testing.T) {
	formal.Check(t, escapeSpec("overlay", realOverlay()))
}

// Each broken variant is a bug the real code could grow; the check must find it, at the named
// property, from the start state where it first shows.
func TestBrokenOverlaysAreFound(t *testing.T) {
	lastOpenTop := func(open overlay.Open) (overlay.Layer, bool) {
		for layer := overlay.Layer(len(open)) - 1; layer >= 0; layer-- {
			if open[layer] {
				return layer, true
			}
		}
		return 0, false
	}
	topSkipsCallLog := func(open overlay.Open) (overlay.Layer, bool) {
		open[overlay.CallLog] = false
		return overlay.Open.Top(open)
	}
	holdsUpToDialog := func(open overlay.Open) bool { return open[overlay.ContextMenu] || open[overlay.Dialog] }

	cases := []struct {
		name         string
		impl         overlayImpl
		wantProperty string // empty: a deadlock
	}{
		{"Top returns the last open layer", overlayImpl{top: lastOpenTop, holds: overlay.Open.HoldsKeyboard}, "TopIsFirstOpen"},
		{"Top never reports the call log", overlayImpl{top: topSkipsCallLog, holds: overlay.Open.HoldsKeyboard}, "TopIsFirstOpen"},
		{"HoldsKeyboard stops at the dialog", overlayImpl{top: overlay.Open.Top, holds: holdsUpToDialog}, "KeyboardHeldByListedLayers"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := formal.Run(escapeSpec("overlay-broken", tc.impl))
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
