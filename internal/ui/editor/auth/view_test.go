package auth

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// authPaneStates are the looks of the Auth pane: one per auth type, the API
// key both with its stored place left empty (the first choice is lit) and
// set to the query, and Bearer with its secret shown.
var authPaneStates = []struct {
	name  string
	setUp func(world *world)
}{
	{"none", func(world *world) {}},
	{"bearer", func(world *world) {
		world.tab.Req.Auth = request.Auth{Type: request.AuthBearer, Token: "example-token"}
	}},
	{"basic", func(world *world) {
		world.tab.Req.Auth = request.Auth{Type: request.AuthBasic, User: "ann", Password: "pw"}
	}},
	{"apikey-header", func(world *world) {
		world.tab.Req.Auth = request.Auth{Type: request.AuthAPIKey, KeyName: "X-Key", KeyValue: "v1"}
	}},
	{"apikey-query", func(world *world) {
		world.tab.Req.Auth = request.Auth{Type: request.AuthAPIKey, KeyName: "X-Key", KeyValue: "v1", KeyIn: request.KeyInQuery}
	}},
	{"bearer-shown", func(world *world) {
		world.tab.Req.Auth = request.Auth{Type: request.AuthBearer, Token: "example-token"}
		world.pane.toggleSecrets(world.tab)
	}},
}

// The Auth pane is pinned as text (element kinds, options, rects) for every
// auth type at the request pane's width and at the narrowest main area.
func TestTheAuthPaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", wideW},
		{"narrow", narrowW},
	}
	for _, state := range authPaneStates {
		for _, size := range sizes {
			name := "authpane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t, &request.Request{Name: "R"})
				state.setUp(world)
				assertDumpPinned(t, world.ctx, world.view(), name, size.width, paneH)
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
		ids         []requestedit.AuthField
		masked      []bool
	}{
		{"bearer", request.Auth{Type: request.AuthBearer}, false, []requestedit.AuthField{"token"}, []bool{true}},
		{"bearer shown", request.Auth{Type: request.AuthBearer}, true, []requestedit.AuthField{"token"}, []bool{false}},
		{"basic", request.Auth{Type: request.AuthBasic}, false, []requestedit.AuthField{"user", "password"}, []bool{false, true}},
		{"api key", request.Auth{Type: request.AuthAPIKey}, false, []requestedit.AuthField{"keyname", "keyvalue"}, []bool{false, true}},
		{"api key shown", request.Auth{Type: request.AuthAPIKey}, true, []requestedit.AuthField{"keyname", "keyvalue"}, []bool{false, false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t, &request.Request{Name: "R", Auth: tc.auth})
			if tc.showSecrets {
				world.pane.toggleSecrets(world.tab)
			}
			fields := behaviorStates(world.view(), "*widget.fieldB")
			if len(fields) != len(tc.ids) {
				t.Fatalf("%d fields, want %d: %q", len(fields), len(tc.ids), fields)
			}
			for i, id := range tc.ids {
				input := world.pane.Field(world.tab, id)
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
			world := newWorld(t, &request.Request{Name: "R", Auth: request.Auth{Type: tc.auth}})
			if tc.showSecrets {
				world.pane.toggleSecrets(world.tab)
			}
			got := behaviorStates(world.view(), "widget.checkB")
			if !slices.Equal(got, tc.want) {
				t.Fatalf("checkboxes %q, want %q", got, tc.want)
			}
		})
	}
}

// Right-clicking an auth field opens the text menu.
func TestTheAuthFieldHasTheTextMenu(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Auth: request.Auth{Type: request.AuthBearer}})
	world.frame(core.Input{RightClick: true}, fieldAt(0))
	menu := widget.PopupMenu(world.ctx)
	if menu == nil || menu.Items[0].Label != "Undo" {
		t.Fatalf("menu = %v", menu)
	}
}

// The API key's place chooser lights Header while nothing is stored and
// stores the clicked choice.
func TestTheAPIKeyPlaceChoiceStoresTheClickedPlace(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Auth: request.Auth{Type: request.AuthAPIKey}})
	world.click("Query")
	if got := world.tab.Req.Auth.KeyIn; got != request.KeyInQuery {
		t.Fatalf("KeyIn = %q, want query", got)
	}
	world.click("Header")
	if got := world.tab.Req.Auth.KeyIn; got != request.KeyInHeader {
		t.Fatalf("KeyIn = %q, want header", got)
	}
}

// Each type chip chooses its auth type.
func TestTheAuthTypeChipsChooseTheType(t *testing.T) {
	cases := []struct {
		label string
		want  request.AuthType
	}{
		{"Bearer", request.AuthBearer},
		{"Basic", request.AuthBasic},
		{"API key", request.AuthAPIKey},
		{"None", request.AuthNone},
	}
	world := newWorld(t, &request.Request{Name: "R"})
	for _, tc := range cases {
		world.click(tc.label)
		if got := world.tab.Req.Auth.Type; got != tc.want {
			t.Fatalf("after %s: type %q, want %q", tc.label, got, tc.want)
		}
	}
}

func TestSecretFieldsAreMaskedUntilShown(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R"})
	world.click("Basic")
	user, password := world.pane.Field(world.tab, "user"), world.pane.Field(world.tab, "password")
	if user.Masked || !password.Masked {
		t.Fatalf("user masked %v, password masked %v", user.Masked, password.Masked)
	}
	world.toggleShowSecrets()
	if world.pane.Field(world.tab, "password").Masked {
		t.Fatal("the toggle must show the secret")
	}
	world.toggleShowSecrets()
	if !world.pane.Field(world.tab, "password").Masked {
		t.Fatal("and hide it again")
	}
}

func TestSwitchingAuthTypesKeepsEachTypesValues(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R"})
	world.click("Bearer")
	world.typeInto(0, "abc")
	world.click("Basic")
	world.click("None")
	world.click("Bearer")
	if world.tab.Req.Auth.Token != "abc" || world.pane.Field(world.tab, "token").Text != "abc" {
		t.Fatalf("auth %+v field %q", world.tab.Req.Auth, world.pane.Field(world.tab, "token").Text)
	}
}
