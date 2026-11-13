package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// longEnvName is wider than the header leaves for the chip in a narrow
// window.
const longEnvName = "Staging environment in the second data centre, read only"

// Widths the header is pinned at: the test window, and one so narrow that
// the chip and the title compete for room.
const (
	envWideW   = 900
	envNarrowW = 280
)

// The header with the environment chip is pinned as text
// (testdata/env-*.dump) for no environment, a short and a long active one,
// at two widths. Only the default state is in a golden scene.
func TestTheEnvironmentChipLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name string
		envs []string
	}{
		{"none", nil},
		{"active", []string{"Dev"}},
		{"active-long", []string{longEnvName}},
	}
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", envWideW},
		{"narrow", envNarrowW},
	}
	for _, tc := range cases {
		for _, size := range sizes {
			name := "env-" + tc.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				for _, env := range tc.envs {
					a.addEnv(env)
				}
				root := a.header()
				Layout(a.ctx, root, size.width, headerH)
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

// Clicking the chip opens the environment menu where the pointer is, with
// a ticked row for the active environment and Rename and Delete only when
// one is active.
func TestClickingTheEnvironmentChipOpensItsMenu(t *testing.T) {
	cases := []struct {
		name   string
		active bool
		chip   string
		want   []widget.MenuItem
	}{
		{"none", false, "No environment", []widget.MenuItem{
			{Label: "✓ No environment"},
			{Label: "   Dev"},
			{Separator: true},
			{Label: "New environment..."},
			{Label: "Rename...", Disabled: true},
			{Label: "Delete", Disabled: true},
		}},
		{"active", true, "Dev", []widget.MenuItem{
			{Label: "   No environment"},
			{Label: "✓ Dev"},
			{Separator: true},
			{Label: "New environment..."},
			{Label: "Rename..."},
			{Label: "Delete"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			a.col.Envs = []model.Environment{{Name: "Dev"}}
			if tc.active {
				a.selectEnv(0)
			}
			clickText(t, a, tc.chip, 0)
			menu := widget.PopupMenu(a.ctx)
			if menu == nil {
				t.Fatal("clicking the chip opened no menu")
			}
			if menu.X <= envWideW/2 || menu.Y <= 0 || menu.Y >= headerH {
				t.Errorf("menu opened at %d,%d, not at the chip", menu.X, menu.Y)
			}
			if len(menu.Items) != len(tc.want) {
				t.Fatalf("menu rows %+v, want %+v", menu.Items, tc.want)
			}
			for i, want := range tc.want {
				got := menu.Items[i]
				if got.Label != want.Label || got.Disabled != want.Disabled || got.Separator != want.Separator {
					t.Errorf("row %d: %q disabled=%v separator=%v, want %q disabled=%v separator=%v",
						i, got.Label, got.Disabled, got.Separator, want.Label, want.Disabled, want.Separator)
				}
			}
		})
	}
}

// Picking an environment in the chip's menu makes it active, and the chip
// shows its name; picking "No environment" goes back.
func TestPickingAnEnvironmentInTheChipsMenuShowsItOnTheChip(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	a.col.Envs = []model.Environment{{Name: "Dev"}, {Name: "Prod"}}
	clickText(t, a, "No environment", 0)
	clickText(t, a, "   Prod", 0)
	if a.env.Index != 1 {
		t.Fatalf("active environment %d, want 1", a.env.Index)
	}
	clickText(t, a, "Prod", 0)
	clickText(t, a, "   No environment", 0)
	if a.env.Index != -1 {
		t.Fatalf("active environment %d, want none", a.env.Index)
	}
	textCenter(t, a, "No environment")
}

// The chip's menu rows that need a name or a yes open a dialog: New and
// Rename a prompt (Rename starting from the active name), Delete a
// confirmation naming the environment.
func TestTheChipsMenuRowsOpenTheirDialogs(t *testing.T) {
	cases := []struct {
		row       string
		title     string
		input     string
		hasInput  bool
		bodyNames string
	}{
		{"New environment...", "New environment", "", true, ""},
		{"Rename...", "Rename environment", "Dev", true, ""},
		{"Delete", "Delete environment", "", false, `Delete "Dev" and its variables?`},
	}
	for _, tc := range cases {
		t.Run(tc.row, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			a.col.Envs = []model.Environment{{Name: "Dev"}}
			a.selectEnv(0)
			clickText(t, a, "Dev", 0)
			clickText(t, a, tc.row, 0)
			if a.dialog == nil {
				t.Fatalf("%s opened no dialog", tc.row)
			}
			if a.dialog.Title != tc.title {
				t.Errorf("title %q, want %q", a.dialog.Title, tc.title)
			}
			if (a.dialog.Input != nil) != tc.hasInput {
				t.Fatalf("input %v, want one: %v", a.dialog.Input, tc.hasInput)
			}
			if tc.hasInput && a.dialog.Input.Text != tc.input {
				t.Errorf("input %q, want %q", a.dialog.Input.Text, tc.input)
			}
			if a.dialog.Body != tc.bodyNames {
				t.Errorf("body %q, want %q", a.dialog.Body, tc.bodyNames)
			}
		})
	}
}

// Hovering the chip says what an environment is.
func TestTheEnvironmentChipHasATooltip(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	x, y := textCenter(t, a, "No environment")
	if got, want := tipAt(a, x, y), "Environment: variables that override the collection's"; got != want {
		t.Fatalf("tooltip %q, want %q", got, want)
	}
}

// textCenter is the middle of the first element of the window that shows
// text exactly.
func textCenter(t *testing.T, a *App, text string) (x, y int) {
	t.Helper()
	needle := fmt.Sprintf("text=%q", text)
	for _, line := range strings.Split(layoutDump(a), "\n") {
		if !strings.Contains(line, needle) {
			continue
		}
		match := rectPattern.FindStringSubmatch(line)
		if match == nil {
			t.Fatalf("no rectangle on %q", line)
		}
		var left, top, width, height int
		fmt.Sscan(match[1], &left)
		fmt.Sscan(match[2], &top)
		fmt.Sscan(match[3], &width)
		fmt.Sscan(match[4], &height)
		return left + width/2, top + height/2
	}
	t.Fatalf("no element with text %q", text)
	return 0, 0
}
