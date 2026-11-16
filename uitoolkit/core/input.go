package core

import (
	"math"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input is one frame of user input.
type Input struct {
	X, Y        int
	Click       bool    // left button went down this frame
	MiddleClick bool    // middle button went down this frame
	MiddleHeld  bool    // middle button is down
	RightClick  bool    // right button went down this frame
	Wheel       float64 // vertical wheel movement, positive = up
	WheelX      float64 // horizontal wheel movement (a touchpad's two-finger swipe), positive = right
	Typed       []rune
	Edits       []Edit
	Enter       bool
	Escape      bool
	Tab         bool

	Held             bool // left button is down
	Shift, Ctrl, Alt bool
	// Ctrl+key shortcuts that went down this frame.
	SelectAll, Copy, Cut, Paste bool
	Undo, Redo                  bool
	Clicks                      int // 1, 2 or 3 for single, double, triple click (with Click)

	// Keys went down this frame, for application shortcuts.
	Keys []ebiten.Key
}

// Scaled converts the pointer position from device to logical pixels.
func (in Input) Scaled(scale float64) Input {
	if !(scale > 0) || math.IsInf(scale, 0) { // int(x/0) is not a position
		return in
	}
	in.X = int(float64(in.X) / scale)
	in.Y = int(float64(in.Y) / scale)
	return in
}

// Pressed reports whether key went down this frame.
func (in Input) Pressed(key ebiten.Key) bool { return slices.Contains(in.Keys, key) }

// Consume removes keys from this frame's input in every form they take
// (the Enter, Escape and Tab flags, the edit of an editing key such as Up
// or Backspace, the Keys list), so a widget under an overlay that used them
// does not act on them too: Enter that accepts a menu item must not type a
// line break.
func (in *Input) Consume(keys ...ebiten.Key) {
	for _, key := range keys {
		switch key {
		case ebiten.KeyEnter:
			in.Enter = false
		case ebiten.KeyEscape:
			in.Escape = false
		case ebiten.KeyTab:
			in.Tab = false
		}
		for _, editKey := range editKeys {
			if editKey.key == key {
				in.Edits = withoutEdit(in.Edits, editKey.edit)
			}
		}
	}
	remaining := make([]ebiten.Key, 0, len(in.Keys))
	for _, down := range in.Keys {
		if !slices.Contains(keys, down) {
			remaining = append(remaining, down)
		}
	}
	in.Keys = remaining
}

func withoutEdit(edits []Edit, drop Edit) []Edit {
	kept := make([]Edit, 0, len(edits))
	for _, edit := range edits {
		if edit != drop {
			kept = append(kept, edit)
		}
	}
	return kept
}

// RestartsCaret reports whether this input shows a text caret again:
// typing, editing keys and clicks do; a pointer moving over the window
// does not.
func (in Input) RestartsCaret() bool {
	return len(in.Typed) > 0 || len(in.Keys) > 0 || len(in.Edits) > 0 || in.Click || in.Enter ||
		in.Paste || in.Cut || in.Undo || in.Redo || in.SelectAll
}

// Hit reports whether a click landed inside r.
func (in Input) Hit(r Rect) bool { return in.Click && r.Contains(in.X, in.Y) }

// Over reports whether the pointer is inside r.
func (in Input) Over(r Rect) bool { return r.Contains(in.X, in.Y) }

var editKeys = []struct {
	key  ebiten.Key
	edit Edit
}{
	{ebiten.KeyBackspace, EditBackspace},
	{ebiten.KeyDelete, EditDelete},
	{ebiten.KeyArrowLeft, EditLeft},
	{ebiten.KeyArrowRight, EditRight},
	{ebiten.KeyHome, EditHome},
	{ebiten.KeyEnd, EditEnd},
	{ebiten.KeyArrowUp, EditUp},
	{ebiten.KeyArrowDown, EditDown},
	{ebiten.KeyPageUp, EditPageUp},
	{ebiten.KeyPageDown, EditPageDown},
}

// KeyRepeat is how a held key repeats: it fires once when pressed, again
// after Delay, then every Interval. The zero value is the built-in timing
// (25 ticks, then every 3), for when the system's own is unknown.
type KeyRepeat struct {
	Delay, Interval time.Duration
}

const (
	defaultRepeatDelayTicks    = 25
	defaultRepeatIntervalTicks = 3
)

// Fires reports whether a key held for the given number of ticks acts on
// this tick, at tps ticks a second. The timing is rounded to whole ticks
// and an interval is at least one tick.
func (repeat KeyRepeat) Fires(held, tps int) bool {
	delay, interval := defaultRepeatDelayTicks, defaultRepeatIntervalTicks
	if repeat.Delay > 0 && repeat.Interval > 0 {
		delay = max(ticks(repeat.Delay, tps), 1)
		interval = max(ticks(repeat.Interval, tps), 1)
	}
	return held == 1 || (held >= delay && (held-delay)%interval == 0)
}

func ticks(duration time.Duration, tps int) int {
	return int((duration*time.Duration(tps) + time.Second/2) / time.Second)
}

func pressed(key ebiten.Key, repeat KeyRepeat) bool {
	return repeat.Fires(inpututil.KeyPressDuration(key), ebiten.TPS())
}

// readInput samples the devices for this frame.
func readInput(repeat KeyRepeat) Input {
	x, y := ebiten.CursorPosition()
	wheelX, wheel := ebiten.Wheel()
	in := Input{
		X: x, Y: y,
		Click:       inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		Wheel:       wheel,
		WheelX:      wheelX,
		Typed:       ebiten.AppendInputChars(nil),
		Enter:       inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		Escape:      inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Tab:         inpututil.IsKeyJustPressed(ebiten.KeyTab),
		Held:        ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		MiddleClick: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle),
		MiddleHeld:  ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle),
		RightClick:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		Keys:        inpututil.AppendJustPressedKeys(nil),
		Shift:       ebiten.IsKeyPressed(ebiten.KeyShift),
		Alt:         ebiten.IsKeyPressed(ebiten.KeyAlt),
		Ctrl:        PrimaryDown(HostOS, ebiten.IsKeyPressed(ebiten.KeyControl), ebiten.IsKeyPressed(ebiten.KeyMeta)),
	}
	if in.Ctrl {
		in.SelectAll = inpututil.IsKeyJustPressed(ebiten.KeyA)
		in.Copy = inpututil.IsKeyJustPressed(ebiten.KeyC)
		in.Cut = inpututil.IsKeyJustPressed(ebiten.KeyX)
		in.Paste = inpututil.IsKeyJustPressed(ebiten.KeyV)
	}
	if in.Ctrl {
		z := inpututil.IsKeyJustPressed(ebiten.KeyZ)
		in.Undo = z && !in.Shift
		in.Redo = inpututil.IsKeyJustPressed(ebiten.KeyY) || (z && in.Shift)
	}
	for _, k := range editKeys {
		if !pressed(k.key, repeat) {
			continue
		}
		in.Edits = append(in.Edits, modifiedEdit(HostOS, k.edit, in.Ctrl, in.Alt))
	}
	return in
}

// wordEdits are the word-wise variants of movement and deletion keys,
// taken while the platform's word modifier is held.
var wordEdits = map[Edit]Edit{
	EditLeft:      EditWordLeft,
	EditRight:     EditWordRight,
	EditBackspace: EditDeleteWordLeft,
	EditDelete:    EditDeleteWordRight,
}

// docEdits are the document-wide variants of Home and End, taken with the
// primary modifier.
var docEdits = map[Edit]Edit{
	EditHome: EditDocStart,
	EditEnd:  EditDocEnd,
}

// macLineEdits are what Command does to the arrows and delete keys in the
// macOS text editors: it acts on the line, since Mac keyboards often have
// no Home and End keys. Elsewhere Control with these keys is word-wise.
var macLineEdits = map[Edit]Edit{
	EditLeft:      EditHome,
	EditRight:     EditEnd,
	EditBackspace: EditDeleteToLineStart,
	EditDelete:    EditDeleteToLineEnd,
}

// modifiedEdit is edit as the held modifiers turn it. Word movement and
// deletion follow each platform's text editors: Option on macOS, where
// Command is the shortcut key and Control belongs to the system, and
// Control elsewhere; Command on macOS makes them line edits and wins over
// Option. primary is the platform's shortcut modifier (PrimaryDown), alt
// the Alt (Option) key.
func modifiedEdit(goos string, edit Edit, primary, alt bool) Edit {
	wordModifier := primary
	if goos == "darwin" {
		wordModifier = alt
		// Return here: Command+Left is the line start, and going on to
		// docEdits would turn the Home it becomes into the document start.
		if line, ok := macLineEdits[edit]; ok && primary {
			return line
		}
	}
	if word, ok := wordEdits[edit]; ok && wordModifier {
		return word
	}
	if doc, ok := docEdits[edit]; ok && primary {
		return doc
	}
	return edit
}

// Multi-click detection: a click within this time and distance of the
// previous one extends the count (double, triple).
const (
	multiClickGap  = 400 * time.Millisecond
	multiClickDist = 4
)

// Sampler reads input each frame and counts rapid clicks.
type Sampler struct {
	// Now is the clock clicks are timed by; nil is time.Now.
	Now func() time.Time
	// Repeat is how held keys repeat; the zero value is the built-in timing.
	Repeat KeyRepeat

	last  time.Time
	x, y  int
	count int
}

// Click registers a click at x, y at time now and returns its count
// (1, 2 or 3; a fourth quick click starts over at 1).
func (s *Sampler) Click(now time.Time, x, y int) int {
	near := abs(x-s.x) <= multiClickDist && abs(y-s.y) <= multiClickDist
	if s.count > 0 && now.Sub(s.last) <= multiClickGap && near && s.count < 3 {
		s.count++
	} else {
		s.count = 1
	}
	s.last, s.x, s.y = now, x, y
	return s.count
}

func (s *Sampler) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Read samples the devices for this frame, including the click count.
func (s *Sampler) Read() Input {
	in := readInput(s.Repeat)
	if in.Click {
		in.Clicks = s.Click(s.now(), in.X, in.Y)
	}
	return in
}
