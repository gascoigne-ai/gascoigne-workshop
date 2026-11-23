package envchip

import (
	"slices"
	"testing"

	envs "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/environments"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fake is every dependency of the chip at once: one collection, and a
// record of what the chip asked of the host.
type fake struct {
	col       *request.Collection // nil: no collection open
	selection envs.Selection
	calls     []string
	prompt    struct {
		title, initial, okLabel string
		onOK                    func(string)
	}
	confirm struct {
		title, body, okLabel string
		onOK                 func()
	}
}

func (fake *fake) Focused() (*request.Collection, envs.Selection, bool) {
	return fake.col, fake.selection, fake.col != nil
}
func (fake *fake) Select(index int) { fake.calls = append(fake.calls, "select "+itoa(index)) }
func (fake *fake) Add(name string)  { fake.calls = append(fake.calls, "add "+name) }
func (fake *fake) Rename(index int, name string) {
	fake.calls = append(fake.calls, "rename "+itoa(index)+" "+name)
}
func (fake *fake) Delete(index int)                             { fake.calls = append(fake.calls, "delete "+itoa(index)) }
func (fake *fake) Open(posX, posY int, items []widget.MenuItem) {}
func (fake *fake) Prompt(title, initial, okLabel string, onOK func(text string)) {
	fake.prompt.title, fake.prompt.initial, fake.prompt.okLabel, fake.prompt.onOK = title, initial, okLabel, onOK
}
func (fake *fake) Confirm(title, body, okLabel string, onOK func()) {
	fake.confirm.title, fake.confirm.body, fake.confirm.okLabel, fake.confirm.onOK = title, body, okLabel, onOK
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// chip is a chip over a fake with environments Dev and Prod, Prod active.
func chip() (*Component, *fake) {
	fake := &fake{
		col:       &request.Collection{Envs: []request.Environment{{Name: "Dev"}, {Name: "Prod"}}},
		selection: envs.Selection{Index: 1},
	}
	return New(Deps{Envs: fake, Menus: fake, Dialogs: fake}), fake
}

// Every row of the menu does its action: a pick selects, New and Rename ask
// for a name first, Delete asks for a yes first, and the separator does
// nothing.
func TestMenuRowsRunTheirAction(t *testing.T) {
	cases := []struct {
		name  string
		row   int
		check func(t *testing.T, fake *fake)
	}{
		{"select none", 0, func(t *testing.T, fake *fake) { wantCalls(t, fake, "select -1") }},
		{"select a named one", 1, func(t *testing.T, fake *fake) { wantCalls(t, fake, "select 0") }},
		{"new asks for a name, then adds", 4, func(t *testing.T, fake *fake) {
			wantCalls(t, fake)
			if fake.prompt.title != "New environment" || fake.prompt.initial != "" || fake.prompt.okLabel != "Create" {
				t.Fatalf("prompt %+v", fake.prompt)
			}
			fake.prompt.onOK("Stage")
			wantCalls(t, fake, "add Stage")
		}},
		{"rename asks, starting with the active name", 5, func(t *testing.T, fake *fake) {
			wantCalls(t, fake)
			if fake.prompt.title != "Rename environment" || fake.prompt.initial != "Prod" || fake.prompt.okLabel != "Rename" {
				t.Fatalf("prompt %+v", fake.prompt)
			}
			fake.prompt.onOK("Live")
			wantCalls(t, fake, "rename 1 Live")
		}},
		{"delete asks, then deletes", 6, func(t *testing.T, fake *fake) {
			wantCalls(t, fake)
			if fake.confirm.title != "Delete environment" || fake.confirm.body != `Delete "Prod" and its variables?` || fake.confirm.okLabel != "Delete" {
				t.Fatalf("confirm %+v", fake.confirm)
			}
			fake.confirm.onOK()
			wantCalls(t, fake, "delete 1")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			component, fake := chip()
			items := component.menu()
			if len(items) != 7 {
				t.Fatalf("menu has %d rows, want 7: %+v", len(items), items)
			}
			if items[tc.row].Run == nil {
				t.Fatalf("row %d has no action", tc.row)
			}
			items[tc.row].Run()
			tc.check(t, fake)
		})
	}
}

// The menu marks the active environment and separates the choices from the
// commands.
func TestMenuMarksTheActiveRowAndSeparates(t *testing.T) {
	component, _ := chip()
	items := component.menu()
	if !items[3].Separator {
		t.Errorf("row 3 %+v, want the separator", items[3])
	}
	for i, item := range items {
		if want := i == 2; (item.Icon == "check") != want {
			t.Errorf("row %d %q icon %q, want checked: %v", i, item.Label, item.Icon, want)
		}
	}
}

// With no collection open there is no menu, and the chip says so.
func TestNoMenuWithoutACollection(t *testing.T) {
	component, fake := chip()
	fake.col = nil
	if items := component.menu(); items != nil {
		t.Fatalf("menu %+v, want none", items)
	}
	if got := component.label(); got != "No environment" {
		t.Fatalf("label %q", got)
	}
}

func wantCalls(t *testing.T, fake *fake, want ...string) {
	t.Helper()
	if !slices.Equal(fake.calls, want) {
		t.Fatalf("calls %v, want %v", fake.calls, want)
	}
}
