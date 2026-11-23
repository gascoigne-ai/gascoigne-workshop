package sizes

import (
	"image/color"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func newCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}
