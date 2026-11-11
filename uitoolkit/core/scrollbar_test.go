package core

import "testing"

func TestScrollbarThumbSizeAndPositionFollowTheContent(t *testing.T) {
	tests := []struct {
		name                      string
		track, content, view, pos int
		wantOff, wantLen          int
		wantOK                    bool
	}{
		{"everything fits", 100, 100, 100, 0, 0, 0, false},
		{"content shorter than the view", 100, 50, 100, 0, 0, 0, false},
		{"empty track", 0, 500, 100, 0, 0, 0, false},
		{"top of a long document has a minimum thumb", 100, 1000, 100, 0, 0, minThumbLen, true},
		{"bottom", 100, 1000, 100, 900, 100 - minThumbLen, minThumbLen, true},
		{"middle", 100, 1000, 100, 450, (100 - minThumbLen) / 2, minThumbLen, true},
		{"proportional length", 100, 200, 100, 0, 0, 50, true},
		{"proportional middle", 100, 200, 100, 50, 25, 50, true},
		{"position past the end is clamped", 100, 200, 100, 999, 50, 50, true},
		{"negative position is clamped", 100, 200, 100, -5, 0, 50, true},
	}
	for _, tt := range tests {
		off, length, ok := Thumb(tt.track, tt.content, tt.view, tt.pos)
		if ok != tt.wantOK || off != tt.wantOff || length != tt.wantLen {
			t.Errorf("%s: got off %d len %d ok %v; want %d %d %v", tt.name, off, length, ok, tt.wantOff, tt.wantLen, tt.wantOK)
		}
	}
}

func TestPosFromOffsetInvertsThumb(t *testing.T) {
	const track, content, view = 100, 1000, 100
	maxPos := content - view
	step := maxPos / (track - minThumbLen) // one pixel of thumb travel
	for _, pos := range []int{0, 100, 450, 899, 900} {
		off, _, _ := Thumb(track, content, view, pos)
		got := PosFromOffset(track, content, view, off)
		if d := got - pos; d < -step || d > step {
			t.Errorf("pos %d -> off %d -> pos %d (tolerance %d)", pos, off, got, step)
		}
	}
	if got := PosFromOffset(track, content, view, 5000); got != maxPos {
		t.Errorf("dragging past the end gives %d, want %d", got, maxPos)
	}
	if got := PosFromOffset(track, content, view, -40); got != 0 {
		t.Errorf("dragging before the start gives %d, want 0", got)
	}
	if got := PosFromOffset(track, 50, view, 10); got != 0 {
		t.Errorf("nothing to scroll gives %d", got)
	}
}

func TestClampScrollKeepsAPositionWithinTheContent(t *testing.T) {
	for _, tt := range []struct{ pos, content, view, want int }{
		{50, 1000, 100, 50}, {-1, 1000, 100, 0}, {950, 1000, 100, 900}, {5, 50, 100, 0},
	} {
		if got := ClampScroll(tt.pos, tt.content, tt.view); got != tt.want {
			t.Errorf("ClampScroll(%d,%d,%d) = %d, want %d", tt.pos, tt.content, tt.view, got, tt.want)
		}
	}
}

func TestPressThumbGrabsTheThumbOrPages(t *testing.T) {
	// content 1000 in a view of 100 on a 100px track: the thumb is 10px.
	grabbed, off, pos := PressThumb(100, 1000, 100, 0, 5)
	if !grabbed || off != 5 || pos != 0 {
		t.Fatalf("on the thumb: %v %d %d", grabbed, off, pos)
	}
	grabbed, _, pos = PressThumb(100, 1000, 100, 0, 60)
	if grabbed || pos != 100 {
		t.Fatalf("below the thumb pages down: %v %d", grabbed, pos)
	}
	grabbed, _, pos = PressThumb(100, 1000, 100, 300, 2)
	if grabbed || pos != 200 {
		t.Fatalf("above the thumb pages up: %v %d", grabbed, pos)
	}
	if grabbed, _, pos = PressThumb(100, 50, 100, 0, 5); grabbed || pos != 0 {
		t.Fatal("content that fits has nothing to grab")
	}
}
