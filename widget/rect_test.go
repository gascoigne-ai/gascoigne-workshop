package widget

import (
	"testing"
	"time"
)

func TestSplitsClamp(t *testing.T) {
	r := Rect{10, 20, 100, 50}
	tests := []struct {
		name       string
		got1, got2 Rect
		want1      Rect
		want2      Rect
	}{
		{"left", first(r.SplitLeft(30)), second(r.SplitLeft(30)), Rect{10, 20, 30, 50}, Rect{40, 20, 70, 50}},
		{"left too big", first(r.SplitLeft(500)), second(r.SplitLeft(500)), r, Rect{110, 20, 0, 50}},
		{"top negative", first(r.SplitTop(-5)), second(r.SplitTop(-5)), Rect{10, 20, 100, 0}, r},
		{"bottom", first(r.SplitBottom(20)), second(r.SplitBottom(20)), Rect{10, 20, 100, 30}, Rect{10, 50, 100, 20}},
		{"right", first(r.SplitRight(25)), second(r.SplitRight(25)), Rect{10, 20, 75, 50}, Rect{85, 20, 25, 50}},
	}
	for _, tt := range tests {
		if tt.got1 != tt.want1 || tt.got2 != tt.want2 {
			t.Errorf("%s: got %+v %+v, want %+v %+v", tt.name, tt.got1, tt.got2, tt.want1, tt.want2)
		}
	}
}

func first(a, _ Rect) Rect  { return a }
func second(_, b Rect) Rect { return b }

func TestContainsAndInset(t *testing.T) {
	r := Rect{10, 10, 20, 20}
	for _, tt := range []struct {
		x, y int
		want bool
	}{{10, 10, true}, {29, 29, true}, {30, 10, false}, {9, 15, false}} {
		if got := r.Contains(tt.x, tt.y); got != tt.want {
			t.Errorf("Contains(%d,%d) = %v", tt.x, tt.y, got)
		}
	}
	if got := r.Inset(30); got.W != 0 || got.H != 0 {
		t.Errorf("Inset beyond size = %+v", got)
	}
}

func TestInputScaled(t *testing.T) {
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
	c, err := NewCtx(Dark())
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

func TestCtrlTurnsMovementIntoWordMovement(t *testing.T) {
	tests := []struct{ in, want Edit }{
		{EditLeft, EditWordLeft}, {EditRight, EditWordRight},
		{EditBackspace, EditDeleteWordLeft}, {EditDelete, EditDeleteWordRight},
	}
	for _, tt := range tests {
		if got, ok := ctrlEdit(tt.in); !ok || got != tt.want {
			t.Errorf("ctrlEdit(%v) = %v %v, want %v", tt.in, got, ok, tt.want)
		}
	}
	if _, ok := ctrlEdit(EditUp); ok {
		t.Error("Up has no Ctrl variant")
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
