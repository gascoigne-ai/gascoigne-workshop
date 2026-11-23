package header

import (
	"os"
	"path/filepath"
	"testing"

	envs "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/environments"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/header/envchip"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fake is every dependency of the banner at once: one collection with the
// given environments, and a count of the settings toggles.
type fake struct {
	col       *request.Collection
	selection envs.Selection
	toggles   int
}

func (fake *fake) Title() string { return fake.col.Name }
func (fake *fake) Toggle()       { fake.toggles++ }
func (fake *fake) Hint(label, action string) string {
	return label + " (Ctrl+,)"
}
func (fake *fake) Focused() (*request.Collection, envs.Selection, bool) {
	return fake.col, fake.selection, true
}
func (fake *fake) Select(int) {}
func (fake *fake) Add(name string) {
	fake.col.Envs = append(fake.col.Envs, request.Environment{Name: name})
	fake.selection.Index = len(fake.col.Envs) - 1
}
func (fake *fake) Rename(int, string)                          {}
func (fake *fake) Delete(int)                                  {}
func (fake *fake) Open(int, int, []widget.MenuItem)            {}
func (fake *fake) Prompt(string, string, string, func(string)) {}
func (fake *fake) Confirm(string, string, string, func())      {}

// banner is a banner over a fake of a collection with no environments.
func banner(t *testing.T) (*Component, *fake, *core.Ctx) {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{col: &request.Collection{}, selection: envs.None()}
	fake.col.Name = "GitHub REST API"
	deps := Deps{Title: fake, Settings: fake, Hints: fake, Env: envchip.Deps{Envs: fake, Menus: fake, Dialogs: fake}}
	return New(deps), fake, ctx
}

// Clicking the gear at the right edge asks the host to toggle the settings.
func TestGearTogglesSettings(t *testing.T) {
	component, fake, ctx := banner(t)
	const width = 900
	click := core.Input{X: width - sizes.HeaderH/2, Y: sizes.HeaderH / 2, Click: true, Held: true}
	for _, in := range []core.Input{click, {X: click.X, Y: click.Y}} {
		ctx.Tick++
		ctx.In = in
		root := component.View(ctx)
		Layout(ctx, root, width, sizes.HeaderH)
		Update(ctx, root)
	}
	if fake.toggles != 1 {
		t.Fatalf("settings toggled %d times, want once", fake.toggles)
	}
}

// Widths the banner is pinned at: the test window, and one so narrow that
// the chip and the title compete for room.
const (
	wideW   = 900
	narrowW = 280
)

// longEnvName is wider than the banner leaves for the chip in a narrow window.
const longEnvName = "Staging environment in the second data centre, read only"

// The banner with the environment chip is pinned as text (testdata, the
// layout the whole window showed before the banner became a component).
func TestTheBannerLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name string
		envs []string
	}{
		{"none", nil},
		{"active", []string{"Dev"}},
		{"active-long", []string{longEnvName}},
	}
	sizesByName := []struct {
		name  string
		width int
	}{{"wide", wideW}, {"narrow", narrowW}}
	for _, tc := range cases {
		for _, size := range sizesByName {
			name := "env-" + tc.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				component, fake, ctx := banner(t)
				for _, env := range tc.envs {
					fake.Add(env)
				}
				root := component.View(ctx)
				Layout(ctx, root, size.width, sizes.HeaderH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}
