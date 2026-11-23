package sizes

import (
	"image/color"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
)

func newCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestScopeHoldsContextSizesAndFixedWidths(t *testing.T) {
	scope := Scope(newCtx(t), nil)
	if scope["wrapChipW"] != 56 {
		t.Errorf("wrapChipW = %v, want 56", scope["wrapChipW"])
	}
	if _, ok := scope["dialogButtonMinW"]; !ok {
		t.Error("dialogButtonMinW is missing")
	}
	for name := range newCtx(t).Sizes.Scope() {
		if _, ok := scope[name]; !ok {
			t.Errorf("context size %q is missing", name)
		}
		break
	}
}

func TestScopeOwnEntriesWin(t *testing.T) {
	scope := Scope(newCtx(t), markup.Scope{"wrapChipW": 1, "mine": 2})
	if scope["wrapChipW"] != 1 || scope["mine"] != 2 {
		t.Fatalf("own entries lost: %v %v", scope["wrapChipW"], scope["mine"])
	}
}
