package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes the request's Settings pane is pinned at: the request pane of the
// test window and the narrowest main area.
const (
	settingsWideW   = winW - sidebarW
	settingsNarrowW = minMainW
	settingsPaneH   = 300
)

// settingsPaneStates are the looks of the pane: the defaults and a settings
// default the hint beside the field follows. Whether a check is ticked is
// not in Dump; TestTheSettingsChecksFollowTheRequest pins it.
var settingsPaneStates = []struct {
	name  string
	setUp func(a *App, tab *reqTab)
}{
	{"defaults", func(a *App, tab *reqTab) {}},
	{"default-timeout", func(a *App, tab *reqTab) {
		a.settings.TimeoutSec = 45
	}},
}

// The request's Settings pane is pinned as text (element kinds, options,
// rectangles) at two widths and its states, so its move to markup can be
// checked to draw the same.
func TestTheRequestSettingsPaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", settingsWideW},
		{"narrow", settingsNarrowW},
	}
	for _, state := range settingsPaneStates {
		for _, size := range sizes {
			name := "settingspane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				tab := a.cur()
				state.setUp(a, tab)
				root := a.settingsPane(settingsPane, tab, &widget.ScrollState{})
				Layout(a.ctx, root, size.width, settingsPaneH)
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

// What Dump does not show of the timeout field: it edits the tab's timeout,
// has no placeholder and does nothing on Enter.
func TestTheTimeoutFieldEditsTheTabsTimeout(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	root := a.settingsPane(settingsPane, tab, &widget.ScrollState{})
	field := findWidget(root, "*widget.fieldB")
	if field == nil {
		t.Fatalf("no field in:\n%s", Dump(root))
	}
	settings := fmt.Sprintf("%+v", field.Behavior())
	for _, want := range []string{"field:" + fmt.Sprintf("%p", &tab.timeout), "placeholder: ", "onEnter:<nil>"} {
		if !strings.Contains(settings, want) {
			t.Errorf("the field lacks %q: %s", want, settings)
		}
	}
}

// Right-clicking the timeout field opens the text menu.
func TestTheTimeoutFieldHasTheTextMenu(t *testing.T) {
	a := openSettingsPane(t)
	t.Cleanup(a.waitWrites)
	rightClick(a, timeoutFieldX, chipsY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}

// Text that is not a whole number of seconds is undone in the field, which
// keeps the keyboard; Dump does not show the field's text.
func TestAnInvalidTimeoutIsUndoneInTheField(t *testing.T) {
	a := openSettingsPane(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	click(a, timeoutFieldX, chipsY)
	a.typeText("7")
	a.typeText("x")
	if tab.timeout.Text != "7" || !tab.timeout.Focused {
		t.Fatalf("field %q focused=%v, want %q still focused", tab.timeout.Text, tab.timeout.Focused, "7")
	}
}

// Dump does not show whether a checkbox is ticked: each one is ticked while
// its behavior is on (redirects followed, the certificate verified).
func TestTheSettingsChecksFollowTheRequest(t *testing.T) {
	cases := []struct {
		name                     string
		noRedirects, insecureTLS bool
		want                     []string
	}{
		{"defaults", false, false, []string{"{Base:{} on:true}", "{Base:{} on:true}"}},
		{"redirects off", true, false, []string{"{Base:{} on:false}", "{Base:{} on:true}"}},
		{"verification off", false, true, []string{"{Base:{} on:true}", "{Base:{} on:false}"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			tab := a.cur()
			tab.Req.NoRedirects, tab.Req.InsecureTLS = tc.noRedirects, tc.insecureTLS
			got := checkStates(a.settingsPane(settingsPane, tab, &widget.ScrollState{}))
			if !slices.Equal(got, tc.want) {
				t.Fatalf("checkboxes %q, want %q", got, tc.want)
			}
		})
	}
}

// checkStates prints the behavior of every checkbox under root, in order.
func checkStates(root Node) []string {
	var states []string
	if fmt.Sprintf("%T", root.Behavior()) == "widget.checkB" {
		states = append(states, fmt.Sprintf("%+v", root.Behavior()))
	}
	for _, child := range root.Children() {
		states = append(states, checkStates(child)...)
	}
	return states
}

// The certificate row is one row and the pane's gap below the redirects row.
const verifyRowY = followRowY + 36

func TestTheVerifyCertificateRowTogglesInsecureTLS(t *testing.T) {
	a := openSettingsPane(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	click(a, followRowX, verifyRowY)
	if !tab.Req.InsecureTLS {
		t.Fatal("the checkbox row must turn verification off")
	}
	click(a, followRowX, verifyRowY)
	if tab.Req.InsecureTLS {
		t.Fatal("a second click must turn it back on")
	}
}
