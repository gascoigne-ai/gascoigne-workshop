package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
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
				a := testApp(t)
				t.Cleanup(a.WaitWrites)
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

// envFitW is a window just wide enough for the title, a chip at its widest
// and the settings button side by side.
const envFitW = 400

// A long environment name used to stretch the chip past the window and
// push the settings button off screen; the chip now stops at a maximum
// width and cuts its name with an ellipsis.
func TestALongEnvironmentNameKeepsTheSettingsButtonInTheWindow(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	a.addEnv(longEnvName)
	root := a.header()
	Layout(a.ctx, root, envFitW, headerH)
	children := root.Children()
	button := children[len(children)-1]
	if right := button.Rect().X + button.Rect().W; right > envFitW {
		t.Errorf("the settings button ends at x=%d, past the %d px window:\n%s", right, envFitW, Dump(root))
	}
	var name *El
	for _, child := range children {
		if strings.Contains(Dump(child), "control=pill") {
			name = child.Children()[0]
		}
	}
	if name == nil {
		t.Fatalf("no chip in the header:\n%s", Dump(root))
	}
	fullW := a.ctx.Width(a.ctx.F.SmallBold, longEnvName)
	if !strings.Contains(Dump(name), "ellipsis=true") || name.Rect().W >= fullW {
		t.Errorf("the chip's name is %d px of its %d, want it cut with an ellipsis: %s", name.Rect().W, fullW, Dump(name))
	}
}

// The header did not fit a 280 px window: the title and the chip kept
// their full widths and pushed the settings button past the edge, even
// with no environment (task 015). Both still show, and "No environment"
// fits there in full.
func TestANarrowWindowKeepsTheSettingsButtonInside(t *testing.T) {
	cases := []struct {
		name  string
		envs  []string
		label string
	}{
		{"none", nil, "No environment"},
		{"long", []string{longEnvName}, longEnvName},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			for _, env := range tc.envs {
				a.addEnv(env)
			}
			root := a.header()
			Layout(a.ctx, root, envNarrowW, headerH)
			children := root.Children()
			button := children[len(children)-1]
			if right := button.Rect().X + button.Rect().W; right > envNarrowW {
				t.Errorf("the settings button ends at x=%d, past the %d px window:\n%s", right, envNarrowW, Dump(root))
			}
			title, chip := children[0], children[2]
			if title.Rect().W == 0 || chip.Rect().W == 0 {
				t.Errorf("title %d px and chip %d px wide, want both shown:\n%s", title.Rect().W, chip.Rect().W, Dump(root))
			}
			label := chip.Children()[0]
			if fullW := a.ctx.Width(a.ctx.F.SmallBold, tc.label); tc.envs == nil && label.Rect().W < fullW {
				t.Errorf("%q is %d of its %d px, want it in full:\n%s", tc.label, label.Rect().W, fullW, Dump(root))
			}
		})
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
			{Label: "No environment", Icon: "check"},
			{Label: "Dev"},
			{Separator: true},
			{Label: "New environment..."},
			{Label: "Rename...", Disabled: true},
			{Label: "Delete", Disabled: true},
		}},
		{"active", true, "Dev", []widget.MenuItem{
			{Label: "No environment"},
			{Label: "Dev", Icon: "check"},
			{Separator: true},
			{Label: "New environment..."},
			{Label: "Rename..."},
			{Label: "Delete"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			a.col().Envs = []request.Environment{{Name: "Dev"}}
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
				if got.Label != want.Label || (got.Icon == "check") != (want.Icon == "check") || got.Disabled != want.Disabled || got.Separator != want.Separator {
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
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	a.col().Envs = []request.Environment{{Name: "Dev"}, {Name: "Prod"}}
	clickText(t, a, "No environment", 0)
	clickText(t, a, "Prod", 0)
	if a.envIndex() != 1 {
		t.Fatalf("active environment %d, want 1", a.envIndex())
	}
	clickText(t, a, "Prod", 0)
	clickText(t, a, "No environment", 0)
	if a.envIndex() != -1 {
		t.Fatalf("active environment %d, want none", a.envIndex())
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
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			a.col().Envs = []request.Environment{{Name: "Dev"}}
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
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	x, y := textCenter(t, a, "No environment")
	if got, want := tipAt(a, x, y), "Environment: variables that override the collection's"; got != want {
		t.Fatalf("tooltip %q, want %q", got, want)
	}
}
