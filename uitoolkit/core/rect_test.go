package core

import (
	"image/color"
	"testing"
	"time"
)

func TestRectContainsItsTopLeftAndNotItsFarEdges(t *testing.T) {
	r := Rect{10, 10, 20, 20}
	for _, tt := range []struct {
		x, y int
		want bool
	}{{10, 10, true}, {29, 29, true}, {30, 10, false}, {9, 15, false}} {
		if got := r.Contains(tt.x, tt.y); got != tt.want {
			t.Errorf("Contains(%d,%d) = %v", tt.x, tt.y, got)
		}
	}
}

func TestInputScaledDividesThePointerByTheScale(t *testing.T) {
	tests := []struct {
		name         string
		x, y         int
		scale        float64
		wantX, wantY int
	}{
		{"identity", 100, 40, 1, 100, 40},
		{"double", 200, 80, 2, 100, 40},
		{"fractional", 300, 150, 1.5, 200, 100},
	}
	for _, tt := range tests {
		got := Input{X: tt.x, Y: tt.y}.Scaled(tt.scale)
		if got.X != tt.wantX || got.Y != tt.wantY {
			t.Errorf("%s: got %d,%d want %d,%d", tt.name, got.X, got.Y, tt.wantX, tt.wantY)
		}
	}
}

// Text measures in logical pixels, so layouts do not shift with the scale.
func TestMeasureIsScaleStable(t *testing.T) {
	c, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	w1, h1 := c.Width(c.F.UI, "Repository Info"), c.LineH(c.F.UI)
	if err := c.SetScale(2); err != nil {
		t.Fatal(err)
	}
	w2, h2 := c.Width(c.F.UI, "Repository Info"), c.LineH(c.F.UI)
	if abs(w1-w2) > 1 || abs(h1-h2) > 1 {
		t.Fatalf("scale 1: %dx%d, scale 2: %dx%d", w1, h1, w2, h2)
	}
	if err := c.SetScale(2); err != nil || c.Scale != 2 {
		t.Fatalf("same scale must be a no-op, err=%v scale=%v", err, c.Scale)
	}
}

func TestTheWordModifierIsOptionOnMacAndControlElsewhere(t *testing.T) {
	tests := []struct {
		name         string
		goos         string
		edit         Edit
		primary, alt bool
		want         Edit
	}{
		{"linux ctrl+left moves a word", "linux", EditLeft, true, false, EditWordLeft},
		{"linux ctrl+right moves a word", "linux", EditRight, true, false, EditWordRight},
		{"linux ctrl+backspace deletes a word", "linux", EditBackspace, true, false, EditDeleteWordLeft},
		{"linux ctrl+delete deletes a word", "linux", EditDelete, true, false, EditDeleteWordRight},
		{"linux alt+left is a plain move", "linux", EditLeft, false, true, EditLeft},
		{"linux ctrl+home goes to the document start", "linux", EditHome, true, false, EditDocStart},
		{"linux ctrl+up has no variant", "linux", EditUp, true, false, EditUp},
		{"windows ctrl+left moves a word", "windows", EditLeft, true, false, EditWordLeft},
		{"windows ctrl+backspace deletes a word", "windows", EditBackspace, true, false, EditDeleteWordLeft},
		{"mac option+left moves a word", "darwin", EditLeft, false, true, EditWordLeft},
		{"mac option+right moves a word", "darwin", EditRight, false, true, EditWordRight},
		{"mac option+backspace deletes a word", "darwin", EditBackspace, false, true, EditDeleteWordLeft},
		{"mac option+delete deletes a word", "darwin", EditDelete, false, true, EditDeleteWordRight},
		{"mac cmd+left moves on the line, not by word", "darwin", EditLeft, true, false, EditHome},
		{"mac cmd+backspace deletes on the line, not by word", "darwin", EditBackspace, true, false, EditDeleteToLineStart},
		{"mac cmd+end goes to the document end", "darwin", EditEnd, true, false, EditDocEnd},
		{"mac option+home stays on the line", "darwin", EditHome, false, true, EditHome},
		{"plain left stays a character move", "darwin", EditLeft, false, false, EditLeft},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := modifiedEdit(test.goos, test.edit, test.primary, test.alt); got != test.want {
				t.Fatalf("modifiedEdit = %v, want %v", got, test.want)
			}
		})
	}
}

// On macOS Command with the arrows and the delete keys acts on the line,
// as in its text editors; elsewhere Control keeps its word meaning and
// Home and End are the line keys.
func TestCommandArrowsAndDeleteKeysAreLineEditsOnMac(t *testing.T) {
	tests := []struct {
		name         string
		goos         string
		edit         Edit
		primary, alt bool
		want         Edit
	}{
		{"mac cmd+left goes to the line start, not the document start", "darwin", EditLeft, true, false, EditHome},
		{"mac cmd+right goes to the line end, not the document end", "darwin", EditRight, true, false, EditEnd},
		{"mac cmd+backspace deletes to the line start", "darwin", EditBackspace, true, false, EditDeleteToLineStart},
		{"mac cmd+delete deletes to the line end", "darwin", EditDelete, true, false, EditDeleteToLineEnd},
		{"mac cmd wins over option", "darwin", EditLeft, true, true, EditHome},
		{"mac cmd+home still goes to the document start", "darwin", EditHome, true, false, EditDocStart},
		{"mac cmd+up has no line variant", "darwin", EditUp, true, false, EditUp},
		{"linux ctrl+left still moves a word", "linux", EditLeft, true, false, EditWordLeft},
		{"linux ctrl+right still moves a word", "linux", EditRight, true, false, EditWordRight},
		{"linux ctrl+backspace still deletes a word", "linux", EditBackspace, true, false, EditDeleteWordLeft},
		{"linux ctrl+delete still deletes a word", "linux", EditDelete, true, false, EditDeleteWordRight},
		{"linux home stays the line start", "linux", EditHome, false, false, EditHome},
		{"windows ctrl+left still moves a word", "windows", EditLeft, true, false, EditWordLeft},
		{"windows ctrl+backspace still deletes a word", "windows", EditBackspace, true, false, EditDeleteWordLeft},
		{"windows ctrl+delete still deletes a word", "windows", EditDelete, true, false, EditDeleteWordRight},
		{"windows end stays the line end", "windows", EditEnd, false, false, EditEnd},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := modifiedEdit(test.goos, test.edit, test.primary, test.alt); got != test.want {
				t.Fatalf("modifiedEdit = %v, want %v", got, test.want)
			}
		})
	}
}

func TestClickCountingNeedsSpeedAndPlace(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	steps := []struct {
		after time.Duration
		x, y  int
		want  int
	}{
		{0, 10, 10, 1},
		{200 * time.Millisecond, 11, 10, 2},  // quick, same place
		{200 * time.Millisecond, 10, 11, 3},  // third
		{200 * time.Millisecond, 10, 10, 1},  // a fourth click starts over
		{900 * time.Millisecond, 10, 10, 1},  // too slow
		{100 * time.Millisecond, 200, 10, 1}, // quick but elsewhere
		{100 * time.Millisecond, 201, 10, 2},
	}
	var s Sampler
	now := t0
	for i, st := range steps {
		now = now.Add(st.after)
		if got := s.Click(now, st.x, st.y); got != st.want {
			t.Errorf("click %d = %d, want %d", i, got, st.want)
		}
	}
}

// Edges land on device pixels and neighbours share them at every scale.
func TestSnapSharesEdgesAndLinesAreWholePixels(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 1.75, 2, 2.5, 3} {
		c, err := NewCtx(testTheme())
		if err != nil {
			t.Fatal(err)
		}
		if err := c.SetScale(scale); err != nil {
			t.Fatal(err)
		}
		for x := 0; x < 200; x++ {
			if v := c.snap(x); v != float32(int(v)) {
				t.Fatalf("scale %v: snap(%d) = %v is not a device pixel", scale, x, v)
			}
			if c.snap(x+1) < c.snap(x) {
				t.Fatalf("scale %v: edges go backwards at %d", scale, x)
			}
		}
		if lw := c.LineWidth(); lw < 1 || lw != float32(int(lw)) {
			t.Fatalf("scale %v: line width %v is not a whole number of pixels", scale, lw)
		}
	}
}

func TestCtrlHomeEndMapToDocMovement(t *testing.T) {
	for in, want := range map[Edit]Edit{EditHome: EditDocStart, EditEnd: EditDocEnd} {
		for _, goos := range []string{"linux", "windows", "darwin"} {
			if got := modifiedEdit(goos, in, true, false); got != want {
				t.Errorf("%s: modifiedEdit(%v) = %v, want %v", goos, in, got, want)
			}
		}
	}
}

// testTheme is the smallest theme a Ctx can be built from; the real
// palettes live in uitoolkit/theming, which imports this package.
func testTheme() Theme {
	return Theme{
		Colors:   map[Token]color.RGBA{Text: RGB(0xffffff), Bg: RGB(0x101010)},
		FontSize: 14, SmallSize: 12, MonoSize: 13,
	}
}
