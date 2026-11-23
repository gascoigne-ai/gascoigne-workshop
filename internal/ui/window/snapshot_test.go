package window

import (
	"image"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestChangesBase(t *testing.T) {
	tests := []struct {
		name    string
		signals frameSignals
		want    bool
	}{
		{"nothing happened", frameSignals{}, false},
		{"the pointer moved", frameSignals{active: true}, false},
		{"a message was drained", frameSignals{drained: true}, true},
		{"a toast expired", frameSignals{pruned: true}, true},
		{"a key event", frameSignals{keys: true}, true},
		{"a handler fired", frameSignals{fired: true}, true},
		{"a step after the build changed the view", frameSignals{stale: true}, true},
		{"every frame is drawn", frameSignals{everyFrame: true}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := changesBase(tc.signals); got != tc.want {
				t.Fatalf("changesBase = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMovedIsActivityOrWhatTheGoroutinesReported(t *testing.T) {
	for _, signals := range []frameSignals{{active: true}, {drained: true}, {pruned: true}} {
		if !signals.moved() {
			t.Errorf("%+v is not moved", signals)
		}
	}
	if (frameSignals{keys: true, fired: true, stale: true}).moved() {
		t.Error("the frame's own results are not activity")
	}
}

// Every field of core.Input is either an event that can change the base (a
// key, a click, text, the wheel) or only the pointer's place and held state.
// core.Input has slices, so it cannot be compared with the zero value; a new
// field fails the count until it is put in a group here.
func TestPointerOnlyInputCoversEveryField(t *testing.T) {
	tests := []struct {
		field string
		in    core.Input
		event bool
	}{
		{"X", core.Input{X: 5}, false},
		{"Y", core.Input{Y: 5}, false},
		{"Held", core.Input{Held: true}, false},
		{"MiddleHeld", core.Input{MiddleHeld: true}, false},
		{"Shift", core.Input{Shift: true}, false},
		{"Ctrl", core.Input{Ctrl: true}, false},
		{"Alt", core.Input{Alt: true}, false},
		{"Clicks", core.Input{Clicks: 2}, false},
		{"Click", core.Input{Click: true}, true},
		{"MiddleClick", core.Input{MiddleClick: true}, true},
		{"RightClick", core.Input{RightClick: true}, true},
		{"Wheel", core.Input{Wheel: -1}, true},
		{"WheelX", core.Input{WheelX: 1}, true},
		{"Typed", core.Input{Typed: []rune("a")}, true},
		{"Edits", core.Input{Edits: []core.Edit{core.EditLeft}}, true},
		{"Enter", core.Input{Enter: true}, true},
		{"Escape", core.Input{Escape: true}, true},
		{"Tab", core.Input{Tab: true}, true},
		{"SelectAll", core.Input{SelectAll: true}, true},
		{"Copy", core.Input{Copy: true}, true},
		{"Cut", core.Input{Cut: true}, true},
		{"Paste", core.Input{Paste: true}, true},
		{"Undo", core.Input{Undo: true}, true},
		{"Redo", core.Input{Redo: true}, true},
		{"Keys", core.Input{Keys: []ebiten.Key{ebiten.KeyF3}}, true},
	}
	fields := reflect.TypeOf(core.Input{}).NumField()
	if fields != len(tests) {
		t.Fatalf("core.Input has %d fields, this table covers %d: put the new one in a group", fields, len(tests))
	}
	for _, tc := range tests {
		if _, ok := reflect.TypeOf(core.Input{}).FieldByName(tc.field); !ok {
			t.Errorf("core.Input has no field %s", tc.field)
		}
		if got := inputEvents(tc.in); got != tc.event {
			t.Errorf("%s: inputEvents = %v, want %v", tc.field, got, tc.event)
		}
	}
}

func TestSnapshotStep(t *testing.T) {
	kept := snapshotGeometry{size: image.Pt(1000, 700), scale: 1, layer: 2}
	with := func(change func(*snapshotGeometry)) snapshotGeometry {
		geometry := kept
		change(&geometry)
		return geometry
	}
	caretKept := with(func(g *snapshotGeometry) { g.caret = caretShown })
	tests := []struct {
		name    string
		phase   snapshotPhase
		kept    snapshotGeometry
		current snapshotGeometry
		layer   int
		want    snapshotAction
	}{
		{"none draws the base once", snapNone, snapshotGeometry{}, kept, 2, snapReallocate},
		{"fresh with the same geometry is reused", snapFresh, kept, kept, 2, snapReuse},
		{"stale draws the base again", snapStale, kept, kept, 2, snapRedraw},
		{"a size change allocates again", snapFresh, kept, with(func(g *snapshotGeometry) { g.size = image.Pt(1200, 700) }), 2, snapReallocate},
		{"a size change of a stale one allocates again", snapStale, kept, with(func(g *snapshotGeometry) { g.size = image.Pt(1200, 700) }), 2, snapReallocate},
		{"a scale change draws again", snapFresh, kept, with(func(g *snapshotGeometry) { g.scale = 1.25 }), 2, snapRedraw},
		{"a layer change draws again", snapFresh, kept, with(func(g *snapshotGeometry) { g.layer = 3 }), 3, snapRedraw},
		{"a base that shows no caret ignores the blink", snapFresh, kept, with(func(g *snapshotGeometry) { g.caret = caretHidden }), 2, snapReuse},
		{"a caret phase flip draws again", snapFresh, caretKept, with(func(g *snapshotGeometry) { g.caret = caretHidden }), 2, snapRedraw},
		{"the same caret phase is reused", snapFresh, caretKept, caretKept, 2, snapReuse},
		{"no modal releases the image", snapFresh, kept, kept, -1, snapRelease},
		{"no modal releases a stale image", snapStale, kept, kept, -1, snapRelease},
		{"no modal and no image releases nothing to do", snapNone, snapshotGeometry{}, kept, -1, snapRelease},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := snapshotStep(tc.phase, tc.kept, tc.current, tc.layer); got != tc.want {
				t.Fatalf("snapshotStep = %v, want %v", got, tc.want)
			}
		})
	}
}
