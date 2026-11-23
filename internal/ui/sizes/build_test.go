package sizes

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Build reads the fixed widths and the context sizes in two layers, so a name
// in both would make the order of the layers matter.
func TestFixedAndCtxSizeNamesAreDisjoint(t *testing.T) {
	for name := range newCtx(t).Sizes.Scope() {
		if _, both := fixedSizes[name]; both {
			t.Errorf("%q is both a fixed width and a context size", name)
		}
	}
}

func TestBuildReadsFixedWidthsAndContextSizes(t *testing.T) {
	ctx := newCtx(t)
	template := markup.Must("t.vml", `<text>{{wrapChipW}} {{rowH}}</text>`)
	want := fmt.Sprintf(`text="%d %d"`, WrapChipW, ctx.Sizes.Row)
	if got := view.Dump(Build(ctx, template, nil)); !strings.Contains(got, want) {
		t.Errorf("dump lacks %s: %s", want, got)
	}
}

func TestOwnShadowsFixedAndCtxNames(t *testing.T) {
	template := markup.Must("t.vml", `<text>{{wrapChipW}} {{rowH}}</text>`)
	got := view.Dump(Build(newCtx(t), template, markup.Scope{"wrapChipW": 1, "rowH": 2}))
	if !strings.Contains(got, `text="1 2"`) {
		t.Errorf("own entries lost: %s", got)
	}
}
