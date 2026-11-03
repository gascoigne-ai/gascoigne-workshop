package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input is one frame of user input.
type Input struct {
	X, Y   int
	Click  bool    // left button went down this frame
	Wheel  float64 // vertical wheel movement, positive = up
	Typed  []rune
	Edits  []Edit
	Enter  bool
	Escape bool
}

// scaled converts the pointer position from device to logical pixels.
func (in Input) scaled(scale float64) Input {
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
		Click:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		Wheel:  wheel,
		Typed:  ebiten.AppendInputChars(nil),
		Enter:  inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		Escape: inpututil.IsKeyJustPressed(ebiten.KeyEscape),
	}
	for _, k := range editKeys {
		if pressed(k.key) {
			in.Edits = append(in.Edits, k.edit)
		}
	}
	return in
}
