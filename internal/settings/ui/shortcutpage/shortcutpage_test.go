package shortcutpage

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

type fakeHost struct{ current settings.Settings }

func (host *fakeHost) Settings() settings.Settings        { return host.current }
func (host *fakeHost) Edit(edit func(*settings.Settings)) { edit(&host.current) }
func (host *fakeHost) TextMenu(*TextField, func()) Option { return OnRightClick(func(int, int) {}) }
func (host *fakeHost) Actions() []Action {
	return []Action{
		{ID: "close-tab", Name: "Close tab", Default: engine.Binding{Key: ebiten.KeyW, Ctrl: true, Bound: true}},
		{ID: "send", Name: "Send request", Default: engine.Binding{Key: ebiten.KeyEnter, Ctrl: true, Bound: true}},
	}
}

func TestSearchMatchesTheActionNameOrTheKey(t *testing.T) {
	tests := []struct {
		query   string
		want    string
		notWant string
	}{
		{"close", "Close tab", "Send request"},
		{"CLOSE", "Close tab", "Send request"},
		{"ctrl+w", "Close tab", "Send request"},
		{"enter", "Send request", "Close tab"},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			var page Page
			page.search = NewTextField(test.query)
			dump := Dump(page.View(&fakeHost{current: settings.Default()}))
			if !strings.Contains(dump, test.want) || strings.Contains(dump, test.notWant) {
				t.Fatalf("search %q:\n%s", test.query, dump)
			}
		})
	}
	var page Page
	page.search = NewTextField("zzzz")
	if !strings.Contains(Dump(page.View(&fakeHost{current: settings.Default()})), "No shortcut matches") {
		t.Fatal("no empty-result note")
	}
}
