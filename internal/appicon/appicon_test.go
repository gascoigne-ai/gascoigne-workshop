package appicon

import "testing"

func TestIconIsRoundedAndHasAMarkAtEverySize(t *testing.T) {
	for _, size := range []int{16, 32, 64, 128, 256} {
		img := Image(size)
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Fatalf("size %d: bounds %v", size, b)
		}
		if a := img.RGBAAt(0, 0).A; a != 0 {
			t.Errorf("size %d: the corner is not transparent (alpha %d)", size, a)
		}
		if a := img.RGBAAt(size/2, size/10).A; a != 0xff {
			t.Errorf("size %d: the top edge middle must be solid, alpha %d", size, a)
		}
		// the mark: pixels near the middle are lighter than the background above it
		mark, bg := img.RGBAAt(size*45/100, size/2), img.RGBAAt(size/2, size*12/100)
		if int(mark.R)+int(mark.G)+int(mark.B) <= int(bg.R)+int(bg.G)+int(bg.B) {
			t.Errorf("size %d: no light mark in the middle (%v on %v)", size, mark, bg)
		}
	}
}
