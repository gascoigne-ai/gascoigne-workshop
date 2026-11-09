package colorpage

import (
	"image/color"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

type fakeHost struct {
	current settings.Settings
	colors  map[w.Token]color.RGBA
}

func (host *fakeHost) Settings() settings.Settings { return host.current }
func (host *fakeHost) Edit(edit func(*settings.Settings)) {
	edit(&host.current)
}
func (host *fakeHost) Color(tok w.Token) color.RGBA {
	if col, ok := host.colors[tok]; ok {
		return col
	}
	return color.RGBA{A: 255}
}
func (host *fakeHost) TextMenu(*TextField, func()) Option { return nil }

func newHost() *fakeHost {
	return &fakeHost{current: settings.Default(), colors: map[w.Token]color.RGBA{}}
}

func firstColor(t *testing.T) appearance.CustomColor {
	t.Helper()
	if len(appearance.CustomColors) == 0 {
		t.Fatal("no custom colors")
	}
	return appearance.CustomColors[0]
}

func TestUnreadableHexIsRefusedWithANote(t *testing.T) {
	var page Page
	host := newHost()
	custom := firstColor(t)
	page.Input(host, custom.Token).Text = "teal"
	page.edited(host, custom)
	if got := page.Note(custom.Token); got != "Use #rrggbb" {
		t.Fatalf("note = %q", got)
	}
	if len(host.current.Colors) != 0 {
		t.Fatalf("a refused color must not be saved: %+v", host.current.Colors)
	}
}

func TestResetAllForgetsOverridesNotesAndFields(t *testing.T) {
	var page Page
	host := newHost()
	custom := firstColor(t)
	host.current.Colors = map[string]map[string]string{appearance.ThemeIDOf(host.current.Theme): {string(custom.Token): "#336699"}}
	page.Input(host, custom.Token).Text = "teal"
	page.edited(host, custom)
	page.resetAll(host)
	if len(host.current.Colors) != 0 || page.Note(custom.Token) != "" || len(page.inputs) != 0 {
		t.Fatalf("colors %+v note %q inputs %d", host.current.Colors, page.Note(custom.Token), len(page.inputs))
	}
}
