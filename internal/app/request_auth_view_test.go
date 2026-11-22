package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// authPaneStates are the looks of the Auth pane: one per auth type, the API
// key both with its stored place left empty (the first choice is lit) and
// set to the query, and Bearer with its secret shown.
var authPaneStates = []struct {
	name  string
	setUp func(tab *reqTab)
}{
	{"none", func(tab *reqTab) {}},
	{"bearer", func(tab *reqTab) {
		tab.Req.Auth = request.Auth{Type: request.AuthBearer, Token: "example-token"}
	}},
	{"basic", func(tab *reqTab) {
		tab.Req.Auth = request.Auth{Type: request.AuthBasic, User: "ann", Password: "pw"}
	}},
	{"apikey-header", func(tab *reqTab) {
		tab.Req.Auth = request.Auth{Type: request.AuthAPIKey, KeyName: "X-Key", KeyValue: "v1"}
	}},
	{"apikey-query", func(tab *reqTab) {
		tab.Req.Auth = request.Auth{Type: request.AuthAPIKey, KeyName: "X-Key", KeyValue: "v1", KeyIn: request.KeyInQuery}
	}},
	{"bearer-shown", func(tab *reqTab) {
		tab.Req.Auth = request.Auth{Type: request.AuthBearer, Token: "example-token"}
		tab.showSecrets = true
	}},
}

// The Auth pane is pinned as text (element kinds, options, rects) for every
// auth type at the request pane's width and at the narrowest main area.
func TestTheAuthPaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", settingsWideW},
		{"narrow", settingsNarrowW},
	}
	for _, state := range authPaneStates {
		for _, size := range sizes {
			name := "authpane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				t.Cleanup(a.WaitWrites)
				tab := a.cur()
				state.setUp(tab)
				root := a.authPane(authPane, tab, &widget.ScrollState{})
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

// What Dump does not show of the text fields: each edits the tab's own
// input for its auth field, with no placeholder and no Enter handler, and
// a secret one is masked until the secrets are shown.
func TestTheAuthFieldsEditTheTabsInputs(t *testing.T) {
	cases := []struct {
		name        string
		auth        request.Auth
		showSecrets bool
		ids         []editor.AuthField
		masked      []bool
	}{
		{"bearer", request.Auth{Type: request.AuthBearer}, false, []editor.AuthField{"token"}, []bool{true}},
		{"bearer shown", request.Auth{Type: request.AuthBearer}, true, []editor.AuthField{"token"}, []bool{false}},
		{"basic", request.Auth{Type: request.AuthBasic}, false, []editor.AuthField{"user", "password"}, []bool{false, true}},
		{"api key", request.Auth{Type: request.AuthAPIKey}, false, []editor.AuthField{"keyname", "keyvalue"}, []bool{false, true}},
		{"api key shown", request.Auth{Type: request.AuthAPIKey}, true, []editor.AuthField{"keyname", "keyvalue"}, []bool{false, false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			tab := a.cur()
			tab.Req.Auth, tab.showSecrets = tc.auth, tc.showSecrets
			fields := behaviorStates(a.authPane(authPane, tab, &widget.ScrollState{}), "*widget.fieldB")
			if len(fields) != len(tc.ids) {
				t.Fatalf("%d fields, want %d: %q", len(fields), len(tc.ids), fields)
			}
			for i, id := range tc.ids {
				input := tab.authInput(id)
				for _, want := range []string{"field:" + fmt.Sprintf("%p", input), "placeholder: ", "onEnter:<nil>"} {
					if !strings.Contains(fields[i], want) {
						t.Errorf("field %s lacks %q: %s", id, want, fields[i])
					}
				}
				if input.Masked != tc.masked[i] {
					t.Errorf("field %s masked %v, want %v", id, input.Masked, tc.masked[i])
				}
			}
		})
	}
}

// Dump does not show whether a checkbox is ticked: Show secrets is ticked
// while the secrets are shown, and a type without secrets has no checkbox.
func TestTheShowSecretsCheckFollowsTheTab(t *testing.T) {
	cases := []struct {
		name        string
		auth        request.AuthType
		showSecrets bool
		want        []string
	}{
		{"none", request.AuthNone, false, nil},
		{"hidden", request.AuthBearer, false, []string{"{Base:{} on:false}"}},
		{"shown", request.AuthBearer, true, []string{"{Base:{} on:true}"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			tab := a.cur()
			tab.Req.Auth.Type, tab.showSecrets = tc.auth, tc.showSecrets
			got := behaviorStates(a.authPane(authPane, tab, &widget.ScrollState{}), "widget.checkB")
			if !slices.Equal(got, tc.want) {
				t.Fatalf("checkboxes %q, want %q", got, tc.want)
			}
		})
	}
}

// Right-clicking an auth field opens the text menu.
func TestTheAuthFieldHasTheTextMenu(t *testing.T) {
	a := openAuthPane(t)
	click(a, authBearerX, chipsY)
	rightClick(a, authFieldX, authRow1Y)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}

// The API key's place chooser lights Header while nothing is stored and
// stores the clicked choice.
func TestTheAPIKeyPlaceChoiceStoresTheClickedPlace(t *testing.T) {
	a := openAuthPane(t)
	click(a, authKeyX, chipsY)
	click(a, 516, authRow3Y) // the Query choice
	if got := a.cur().Req.Auth.KeyIn; got != request.KeyInQuery {
		t.Fatalf("KeyIn = %q, want query", got)
	}
	click(a, 440, authRow3Y) // the Header choice
	if got := a.cur().Req.Auth.KeyIn; got != request.KeyInHeader {
		t.Fatalf("KeyIn = %q, want header", got)
	}
}

// Each type chip chooses its auth type.
func TestTheAuthTypeChipsChooseTheType(t *testing.T) {
	cases := []struct {
		name string
		x    int
		want request.AuthType
	}{
		{"bearer", authBearerX, request.AuthBearer},
		{"basic", authBasicX, request.AuthBasic},
		{"api key", authKeyX, request.AuthAPIKey},
		{"none", authNoneX, request.AuthNone},
	}
	a := openAuthPane(t)
	for _, tc := range cases {
		click(a, tc.x, chipsY)
		if got := a.cur().Req.Auth.Type; got != tc.want {
			t.Fatalf("after %s: type %q, want %q", tc.name, got, tc.want)
		}
	}
}
