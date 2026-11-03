package widget

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input is one frame of user input.
type Input struct {
	X, Y        int
	Click       bool    // left button went down this frame
	MiddleClick bool    // middle button went down this frame
	RightClick  bool    // right button went down this frame
	Wheel       float64 // vertical wheel movement, positive = up
	Typed       []rune
	Edits       []Edit
	Enter       bool
	Escape      bool
	Tab         bool

	Held        bool // left button is down
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
	in.X = int(float64(in.X) / scale)
	in.Y = int(float64(in.Y) / scale)
	return in
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

// Key repeat: after repeatDelay frames held, fire every repeatEvery.
const (
	repeatDelay = 25
	repeatEvery = 3
)

func pressed(key ebiten.Key) bool {
	d := inpututil.KeyPressDuration(key)
	return d == 1 || (d >= repeatDelay && d%repeatEvery == 0)
}

// ReadInput samples the devices for this frame.
func ReadInput() Input {
	x, y := ebiten.CursorPosition()
	_, wheel := ebiten.Wheel()
	in := Input{
		X: x, Y: y,
		Click:       inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		Wheel:       wheel,
		Typed:       ebiten.AppendInputChars(nil),
		Enter:       inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		Escape:      inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Tab:         inpututil.IsKeyJustPressed(ebiten.KeyTab),
		Held:        ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		MiddleClick: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle),
		RightClick:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		Keys:        inpututil.AppendJustPressedKeys(nil),
		Shift:       ebiten.IsKeyPressed(ebiten.KeyShift),
		Alt:         ebiten.IsKeyPressed(ebiten.KeyAlt),
		Ctrl:        ebiten.IsKeyPressed(ebiten.KeyControl),
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
		if !pressed(k.key) {
			continue
		}
		edit := k.edit
		if w, ok := ctrlEdit(edit); ok && in.Ctrl {
			edit = w
		}
		in.Edits = append(in.Edits, edit)
	}
	return in
}

// ctrlEdits turns movement and deletion keys into their word-wise
// variants when Ctrl is held.
var ctrlEdits = map[Edit]Edit{
	EditLeft:      EditWordLeft,
	EditRight:     EditWordRight,
	EditBackspace: EditDeleteWordLeft,
	EditDelete:    EditDeleteWordRight,
	EditHome:      EditDocStart,
	EditEnd:       EditDocEnd,
}

func ctrlEdit(e Edit) (Edit, bool) {
	w, ok := ctrlEdits[e]
	return w, ok
}

// Multi-click detection: a click within this time and distance of the
// previous one extends the count (double, triple).
const (
	multiClickGap  = 400 * time.Millisecond
	multiClickDist = 4
)

// Sampler reads input each frame and counts rapid clicks.
type Sampler struct {
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

// Read samples the devices for this frame, including the click count.
func (s *Sampler) Read() Input {
	in := ReadInput()
	if in.Click {
		in.Clicks = s.Click(time.Now(), in.X, in.Y)
	}
	return in
}
