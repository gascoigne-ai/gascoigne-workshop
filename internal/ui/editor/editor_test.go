package editor

import (
	"slices"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/assertions"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/auth"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/body"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/graphql"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/grpc"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/urlbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// host answers every child's Tabs (and the tables' rows) over real tabs. No
// table is visible, so a table in bulk mode counts as left.
type host struct {
	set  *tabs.Set
	rows []request.KV
}

func (fake *host) Current() *tabs.Tab { return fake.set.Current() }
func (fake *host) IsOpen(tab *tabs.Tab) bool {
	return slices.Contains(fake.set.Tabs(), tab)
}
func (fake *host) Visible() (logic.Table, bool)   { return logic.Table{}, false }
func (fake *host) Rows(logic.Table) *[]request.KV { return &fake.rows }
func (fake *host) Changed(logic.Table)            {}

// newEditor is the composite over two open tabs, the first current.
func newEditor(t *testing.T) (*Component, *host) {
	t.Helper()
	fake := &host{set: tabs.NewSet()}
	fake.set.Open(&request.Request{Name: "first", URL: "https://one.test"})
	fake.set.Open(&request.Request{Name: "second", URL: "https://two.test"})
	fake.set.Apply(tabs.Activate{Index: 0})
	editor := New(Deps{
		URLBar:     urlbar.Deps{Tabs: fake},
		Body:       body.Deps{Tabs: fake},
		GraphQL:    graphql.Deps{Tabs: fake},
		Auth:       auth.Deps{Tabs: fake},
		Assertions: assertions.Deps{Tabs: fake},
		GRPC:       grpc.Deps{Tabs: fake},
		MCP:        mcp.Deps{Tabs: fake},
		WS:         ws.Deps{Tabs: fake},
		Tables:     kvtable.Deps{Tables: fake, Tabs: fake},
	})
	return editor, fake
}

func TestTypingIsTodaysTextFields(t *testing.T) {
	tests := []struct {
		name  string
		focus func(editor *Component, tab *tabs.Tab)
		want  bool
	}{
		{"nothing focused", func(*Component, *tabs.Tab) {}, false},
		{"url field", func(editor *Component, tab *tabs.Tab) { editor.URLBar.Field(tab).Focused = true }, true},
		{"body area", func(editor *Component, tab *tabs.Tab) { editor.Body.Area(tab).Focused = true }, true},
		{"body path field", func(editor *Component, tab *tabs.Tab) { editor.Body.File(tab).Focused = true }, true},
		{"socket composer", func(editor *Component, tab *tabs.Tab) { editor.WS.Composer(tab).Focused = true }, true},
		{"auth field", func(editor *Component, tab *tabs.Tab) { editor.Auth.Field(tab, requestedit.AuthToken).Focused = true }, true},
		{"table cell", func(editor *Component, tab *tabs.Tab) {
			editor.Tables.BeginEdit(logic.Table{Tab: tab, Kind: logic.Params}, 0, 0)
		}, true},
		{"gRPC path fields are not counted", func(editor *Component, tab *tabs.Tab) {
			if field := editor.GRPC.Input(tab, requestedit.ProtoFiles, 0); field != nil {
				field.Focused = true
			}
		}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			editor, fake := newEditor(t)
			test.focus(editor, fake.set.Current())
			if got := editor.Typing(); got != test.want {
				t.Errorf("Typing() = %v, want %v", got, test.want)
			}
		})
	}
}

// Only the current tab's fields count: a focused field of a tab that is not
// showing does not hold back the shortcuts.
func TestTypingIgnoresAFieldOfAnotherTab(t *testing.T) {
	editor, fake := newEditor(t)
	editor.URLBar.Field(fake.set.Tabs()[1]).Focused = true
	if editor.Typing() {
		t.Error("Typing() is true for a field of a tab that is not current")
	}
}

func TestTickTicksEveryChild(t *testing.T) {
	editor, fake := newEditor(t)
	closing := fake.set.Tabs()[1]
	editor.URLBar.Field(closing).Text = "draft"
	editor.Body.Area(closing).Text = "draft"
	editor.WS.Composer(closing).Text = "draft"
	editor.Auth.Field(closing, requestedit.AuthToken).Text = "draft"
	editor.Tables.ToggleBulk(logic.Table{Tab: closing, Kind: logic.Params}) // reports a change on Tick: its table is not showing
	fake.set.Close(1)

	if !editor.Tick() {
		t.Error("Tick() = false, want the tables' change reported")
	}
	got := map[string]string{
		"urlbar": editor.URLBar.Field(closing).Text,
		"body":   editor.Body.Area(closing).Text,
		"ws":     editor.WS.Composer(closing).Text,
		"auth":   editor.Auth.Field(closing, requestedit.AuthToken).Text,
	}
	for name, text := range got {
		if text == "draft" {
			t.Errorf("%s kept the state of a closed tab", name)
		}
	}
}

func TestKeysClaimURLThenBody(t *testing.T) {
	tests := []struct {
		name  string
		focus func(editor *Component, tab *tabs.Tab)
		in    core.Input
		want  bool
	}{
		{"nothing focused", func(*Component, *tabs.Tab) {}, core.Input{Undo: true}, false},
		{"url field takes undo", func(editor *Component, tab *tabs.Tab) { editor.URLBar.Field(tab).Focused = true }, core.Input{Undo: true}, true},
		{"body takes redo", func(editor *Component, tab *tabs.Tab) { editor.Body.Area(tab).Focused = true }, core.Input{Redo: true}, true},
		{"both focused", func(editor *Component, tab *tabs.Tab) {
			editor.URLBar.Field(tab).Focused = true
			editor.Body.Area(tab).Focused = true
		}, core.Input{Undo: true}, true},
		{"escape is nobody's", func(editor *Component, tab *tabs.Tab) { editor.URLBar.Field(tab).Focused = true }, core.Input{Escape: true}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			editor, fake := newEditor(t)
			test.focus(editor, fake.set.Current())
			if got := editor.Keys(test.in); got != test.want {
				t.Errorf("Keys() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCommandsAreTheTables(t *testing.T) {
	editor, _ := newEditor(t)
	ids := func(commands []keymap.Command) []keymap.ID {
		var out []keymap.ID
		for _, command := range commands {
			out = append(out, command.ID)
		}
		return out
	}
	want := ids(editor.Tables.Commands())
	if len(want) == 0 {
		t.Fatal("the tables add no commands, the test pins nothing")
	}
	if got := ids(editor.Commands()); !slices.Equal(got, want) {
		t.Errorf("Commands() = %v, want %v", got, want)
	}
}
