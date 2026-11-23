package fontlist

import (
	"image/color"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// fakeHost keeps the settings in memory; Send applies a Change as the
// application would.
type fakeHost struct {
	current settings.Settings
}

func newHost(t *testing.T) *fakeHost {
	return &fakeHost{current: settings.Default()}
}

func (h *fakeHost) Size() (int, int)                   { return 1280, 800 }
func (h *fakeHost) Settings() settings.Settings        { return h.current }
func (h *fakeHost) Pointer() (int, int)                { return 400, 300 }
func (h *fakeHost) FontStore() FontStore               { return fonts.System{} }
func (h *fakeHost) SetFonts(ui, code w.FontData) error { return nil }

func (h *fakeHost) Send(msg prefs.Msg) {
	switch msg := msg.(type) {
	case prefs.Change:
		msg.Edit(&h.current)
	case prefs.Later:
		msg.Func()
	}
}

func dump(node Node) string { return Dump(node) }

// testTheme is the theme the views are built with in tests.
func testTheme() core.Theme {
	return core.Theme{
		Colors:   map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff), core.Bg: core.RGB(0x101010)},
		FontSize: 14, SmallSize: 12, MonoSize: 13,
	}
}

// testCtx is the context the views are built with in tests: the default
// faces, so the sizes are the default ones.
func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}
