package core

import "testing"

func TestWidthMeasuresAStringOncePerFace(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	first := ctx.Width(ctx.F.UI, "Send request")
	second := ctx.Width(ctx.F.UI, "Send request")
	if first != second || first <= 0 {
		t.Fatalf("widths %d and %d, want the same positive width", first, second)
	}
	if ctx.Stats.Widths != 1 {
		t.Fatalf("measured %d times, want once", ctx.Stats.Widths)
	}
	if bold := ctx.Width(ctx.F.Bold, "Send request"); bold == first {
		t.Fatalf("bold and regular share the width %d: the face is not part of the cache key", bold)
	}
	if ctx.Stats.Widths != 2 {
		t.Fatalf("measured %d times after a second face, want 2", ctx.Stats.Widths)
	}
}

// A face is a new pointer whenever an element picks its own font, so the
// cache must key on what the face is made of, not on the pointer.
func TestWidthCacheHitsForAnEqualFaceAtAnotherAddress(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	copied := *ctx.F.UI
	ctx.Width(ctx.F.UI, "Headers")
	ctx.Width(&copied, "Headers")
	if ctx.Stats.Widths != 1 {
		t.Fatalf("measured %d times, want once for an equal face", ctx.Stats.Widths)
	}
}

// A width is divided by the scale, so one kept from another scale is wrong.
func TestWidthCacheIsClearedWhenTheScaleChanges(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	ctx.Width(ctx.F.UI, "Collections")
	if err := ctx.SetScale(2); err != nil {
		t.Fatal(err)
	}
	ctx.Width(ctx.F.UI, "Collections")
	if ctx.Stats.Widths != 2 {
		t.Fatalf("measured %d times across a scale change, want 2", ctx.Stats.Widths)
	}
}

func TestWidthCacheStaysBounded(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 * widthCacheLimit {
		ctx.Width(ctx.F.UI, string(rune('a'+i%26))+string(rune(0x100+i)))
	}
	if got := len(ctx.widths); got > widthCacheLimit {
		t.Fatalf("%d cached widths, want at most %d", got, widthCacheLimit)
	}
}
