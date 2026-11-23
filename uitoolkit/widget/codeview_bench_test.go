package widget

import (
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
)

// The CodeView benchmarks time the cost of a large response body (the view
// limit is 2 MiB, about 40k lines of pretty-printed JSON) on the frames that
// change nothing: an Update with no input and a Draw of the same rows.

const benchLines = 40000

func benchCodeView(b *testing.B, wrap bool) (*CodeView, *core.Ctx, core.Rect) {
	b.Helper()
	ctx, err := core.NewCtx(theming.Dark())
	if err != nil {
		b.Fatal(err)
	}
	lines := make([]Line, benchLines)
	for i := range lines {
		lines[i] = Line{Spans: []Span{
			{Text: "    \"field" + strconv.Itoa(i) + "\": "},
			{Text: "\"a value of some length that wraps in a narrow view " + strconv.Itoa(i) + "\","},
		}}
	}
	view := &CodeView{Wrap: wrap}
	view.SetLines(lines)
	rect := core.Rect{W: 700, H: 500}
	view.Update(ctx, rect)
	view.ScrollY = 200000
	view.Update(ctx, rect)
	b.ReportAllocs()
	b.ResetTimer()
	return view, ctx, rect
}

func BenchmarkCodeViewUpdate(b *testing.B) {
	for _, wrap := range []bool{false, true} {
		b.Run("wrap="+strconv.FormatBool(wrap), func(b *testing.B) {
			view, ctx, rect := benchCodeView(b, wrap)
			for i := 0; i < b.N; i++ {
				view.Update(ctx, rect)
			}
		})
	}
}

func BenchmarkCodeViewDraw(b *testing.B) {
	for _, wrap := range []bool{false, true} {
		b.Run("wrap="+strconv.FormatBool(wrap), func(b *testing.B) {
			view, ctx, rect := benchCodeView(b, wrap)
			screen := ebiten.NewImage(rect.W, rect.H)
			for i := 0; i < b.N; i++ {
				view.Draw(ctx, screen, rect)
			}
		})
	}
}
