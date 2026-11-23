package core

import (
	"testing"
)

func TestAdvanceIsMeasuredOncePerFont(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	first := ctx.Advance(ctx.F.Mono)
	copied := *ctx.F.Mono
	second := ctx.Advance(&copied)
	if first != second || first <= 0 {
		t.Fatalf("advances %v and %v, want the same positive advance", first, second)
	}
	if ctx.Stats.Advances != 1 {
		t.Fatalf("measured %d times, want once for an equal face", ctx.Stats.Advances)
	}
	ctx.Advance(ctx.F.UI)
	if ctx.Stats.Advances != 2 {
		t.Fatalf("measured %d times after a second face, want 2", ctx.Stats.Advances)
	}
}

// An advance is divided by the scale and set by the face, so one kept
// across any of these changes would be wrong.
func TestAdvanceIsMeasuredAgainAfterScaleThemeOrFontsChange(t *testing.T) {
	tests := []struct {
		name   string
		change func(ctx *Ctx) error
	}{
		{"scale", func(ctx *Ctx) error { return ctx.SetScale(2) }},
		{"theme", func(ctx *Ctx) error { return ctx.SetTheme(testTheme()) }},
		{"fonts", func(ctx *Ctx) error { return ctx.SetFonts(FontData{}, FontData{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewCtx(testTheme())
			if err != nil {
				t.Fatal(err)
			}
			ctx.Stats = &DrawStats{}
			ctx.Advance(ctx.F.Mono)
			if err := tt.change(ctx); err != nil {
				t.Fatal(err)
			}
			ctx.Advance(ctx.F.Mono)
			if ctx.Stats.Advances != 2 {
				t.Fatalf("measured %d times across a %s change, want 2", ctx.Stats.Advances, tt.name)
			}
		})
	}
}
