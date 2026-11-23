package shortcutpage

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

type fakeHost struct {
	current settings.Settings
	os      string // "" is linux
}

func (host *fakeHost) HostOS() string {
	if host.os == "" {
		return "linux"
	}
	return host.os
}

func (host *fakeHost) Settings() settings.Settings        { return host.current }
func (host *fakeHost) Edit(edit func(*settings.Settings)) { edit(&host.current) }
func (host *fakeHost) Keymap() keymap.Keymap {
	commands := []keymap.Command{
		{ID: "close-tab", Name: "Close tab", Default: keymap.Chord{Key: int(ebiten.KeyW), Ctrl: true}},
		{ID: "send", Name: "Send request", Default: keymap.Chord{Key: int(ebiten.KeyEnter), Ctrl: true}},
	}
	return keymap.Resolve(host.current.Shortcuts, commands, ParseChord)
}

func TestChordRoundTripsThroughBinding(t *testing.T) {
	for _, text := range []string{"Ctrl+W", "Ctrl+Shift+Tab", "Alt+Up", "Delete", "A", "Ctrl+A"} {
		binding, err := engine.ParseBinding(text)
		if err != nil {
			t.Fatal(err)
		}
		chord, bound := ChordOf(binding)
		if !bound || BindingOf(chord) != binding {
			t.Errorf("%s: chord %+v bound %v gives back %+v, want %+v", text, chord, bound, BindingOf(chord), binding)
		}
		parsed, parsedBound, err := ParseChord(text)
		if err != nil || !parsedBound || parsed != chord {
			t.Errorf("%s: ParseChord = %+v %v %v, want %+v", text, parsed, parsedBound, err, chord)
		}
	}
	if _, bound := ChordOf(engine.Binding{}); bound {
		t.Error("an unbound binding has no chord")
	}
	if _, bound, err := ParseChord(""); bound || err != nil {
		t.Errorf("empty text is unbound without error, got %v %v", bound, err)
	}
	if _, _, err := ParseChord("Ctrl+Nonsense"); err == nil {
		t.Error("nonsense must not parse")
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
			dump := Dump(page.View(newCtx(t), &fakeHost{current: settings.Default()}))
			if !strings.Contains(dump, test.want) || strings.Contains(dump, test.notWant) {
				t.Fatalf("search %q:\n%s", test.query, dump)
			}
		})
	}
	var page Page
	page.search = NewTextField("zzzz")
	if !strings.Contains(Dump(page.View(newCtx(t), &fakeHost{current: settings.Default()})), "No shortcut matches") {
		t.Fatal("no empty-result note")
	}
}

func (host *fakeHost) Send(msg prefs.Msg) {
	if edit, ok := msg.(prefs.Edit); ok {
		host.Edit(edit.Edit)
	}
}

func TestKeyTextFollowsTheHostOfEachPage(t *testing.T) {
	tests := []struct{ os, want string }{{"darwin", "Cmd+Enter"}, {"linux", "Ctrl+Enter"}}
	for _, test := range tests {
		t.Run(test.os, func(t *testing.T) {
			t.Parallel()
			time.Sleep(20 * time.Millisecond) // let both cases overlap
			host := &fakeHost{current: settings.Default(), os: test.os}
			if got := KeyText(host, "send"); got != test.want {
				t.Fatalf("KeyText = %q, want %q", got, test.want)
			}
		})
	}
}
