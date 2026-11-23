package welcome

import (
	"strings"
	"testing"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fakeDialogs records what the component asks the window to show.
type fakeDialogs struct {
	state   widget.DialogState
	content func(*w.Ctx) Node
	width   int
	opened  int
	closed  int
}

func (fake *fakeDialogs) Open(state widget.DialogState, content func(*w.Ctx) Node, width int) func() {
	fake.state, fake.content, fake.width = state, content, width
	fake.opened++
	return func() { fake.closed++ }
}

type fakeWindow struct{ width, height int }

func (fake fakeWindow) Size() (int, int) { return fake.width, fake.height }

func newFake() (*Component, *fakeDialogs) {
	dialogs := &fakeDialogs{}
	return New(Deps{Dialogs: dialogs, Window: fakeWindow{1280, 800}}), dialogs
}

func TestOpenShowsTheNotesInAWideDialog(t *testing.T) {
	c, dialogs := newFake()
	c.Open()
	if dialogs.opened != 1 || dialogs.width != welcomeW {
		t.Fatalf("opened %d at width %d, want one dialog %d wide", dialogs.opened, dialogs.width, welcomeW)
	}
	if !strings.Contains(dialogs.state.Title, "0.1.0 alpha") || dialogs.state.Body != welcomeLead {
		t.Fatalf("title %q body %q", dialogs.state.Title, dialogs.state.Body)
	}
	if len(dialogs.state.Buttons) != 1 || !dialogs.state.Buttons[0].Primary || dialogs.state.Buttons[0].Label != "Get started" {
		t.Fatalf("buttons %+v, want the one primary Get started", dialogs.state.Buttons)
	}
	dialogs.state.Buttons[0].Run()
	if dialogs.closed != 1 {
		t.Fatal("Get started did not close the dialog")
	}
	if dialogs.content == nil {
		t.Fatal("the dialog has no content")
	}
	if dump := Dump(dialogs.content(nil)); !strings.Contains(dump, "Early access") {
		t.Fatalf("the content does not show the notes:\n%s", dump)
	}
}

// The notes open at their top every time, not where they were left.
func TestWelcomeContentScrollsFromTheTopEachTimeItOpens(t *testing.T) {
	c, _ := newFake()
	c.Open()
	c.scroll.Y = 80
	c.Open()
	if c.scroll.Y != 0 {
		t.Fatalf("scroll %v on a fresh welcome", c.scroll.Y)
	}
}

func TestWelcomeNotesNameEveryProtocol(t *testing.T) {
	var all strings.Builder
	for _, section := range welcomeNotes {
		all.WriteString(section.Title + "\n" + strings.Join(section.Items, "\n") + "\n")
	}
	for _, name := range []string{"REST", "WebSocket", "SSE", "MCP", "gRPC", "cURL", "OpenAPI", "Postman"} {
		if !strings.Contains(all.String(), name) {
			t.Errorf("the welcome notes do not mention %s", name)
		}
	}
}

// The notes are prose, and a Lines column does not wrap, so the dialog
// wraps each point itself at a word boundary.
func TestWrapBreaksAtWordsWithinTheLimit(t *testing.T) {
	tests := []struct {
		name, text string
		limit      int
		want       string
	}{
		{"short text stays", "one two", 20, "one two"},
		{"breaks at the last space that fits", "one two three four", 9, "one two\nthree\nfour"},
		{"a word longer than the limit stands alone", "tiny enormousword end", 6, "tiny\nenormousword\nend"},
		{"empty text", "", 10, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := wrap(test.text, test.limit); got != test.want {
				t.Fatalf("wrap(%q, %d) = %q, want %q", test.text, test.limit, got, test.want)
			}
		})
	}
}
