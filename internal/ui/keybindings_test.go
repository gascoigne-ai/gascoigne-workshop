package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

func TestEveryDefaultBindingRoundTripsAndHasAnID(t *testing.T) {
	ids := map[string]bool{}
	bindings := map[string]string{}
	for _, s := range shortcuts {
		if s.id == "" || ids[s.id] {
			t.Errorf("shortcut %q has a missing or duplicate id %q", s.name, s.id)
		}
		ids[s.id] = true
		def := engine.Binding{Key: s.key, Ctrl: s.ctrl, Shift: s.shift, Alt: s.alt, Bound: true}
		back, err := engine.ParseBinding(def.String())
		if err != nil || back != def {
			t.Errorf("%s: %q does not round trip: %+v %v", s.id, def.String(), back, err)
		}
		if other, dup := bindings[def.String()]; dup {
			t.Errorf("%s and %s share %s", s.id, other, def.String())
		}
		bindings[def.String()] = s.id
	}
}

func byID(id string) shortcut {
	for _, s := range shortcuts {
		if s.id == id {
			return s
		}
	}
	panic("no shortcut " + id)
}

func TestEffectiveBindingsLayerTheUsersOverToDefaults(t *testing.T) {
	a, _ := newTestApp(t)
	closeTab := byID("close-tab")
	def := a.bindingOf(closeTab)
	if def.String() != "Ctrl+W" {
		t.Fatalf("default = %q", def)
	}
	for _, tt := range []struct {
		name     string
		override map[string]string
		want     string
	}{
		{"a valid override replaces it", map[string]string{"close-tab": "Ctrl+Q"}, "Ctrl+Q"},
		{"an empty string unbinds", map[string]string{"close-tab": ""}, ""},
		{"an unparsable value falls back to the default", map[string]string{"close-tab": "Banana"}, "Ctrl+W"},
		{"other actions' overrides do not matter", map[string]string{"new-request": "Ctrl+N"}, "Ctrl+W"},
		{"unknown actions are ignored", map[string]string{"teleport": "Ctrl+J"}, "Ctrl+W"},
	} {
		a.settings.Shortcuts = tt.override
		if got := a.bindingOf(closeTab).String(); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
	a.settings.Shortcuts = map[string]string{"close-tab": ""}
	if a.bindingOf(closeTab).Bound {
		t.Error("an unbound action is not bound")
	}
}

func TestCheckBinding(t *testing.T) {
	a, _ := newTestApp(t)
	mk := func(s string) engine.Binding {
		b, err := engine.ParseBinding(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	tests := []struct {
		name    string
		id      string
		combo   string
		wantErr string // substring; "" means allowed
	}{
		{"a free combination", "close-tab", "Ctrl+Q", ""},
		{"an action's own binding", "close-tab", "Ctrl+W", ""},
		{"taken by another action, named", "close-tab", "Ctrl+T", "New request"},
		{"taken, with shift", "close-tab", "Ctrl+Shift+Tab", "Previous tab"},
		{"the text widgets' clipboard keys", "close-tab", "Ctrl+C", "text"},
		{"undo", "close-tab", "Ctrl+Z", "text"},
		{"select all", "close-tab", "Ctrl+A", "text"},
		{"a plain letter is allowed, with a warning", "close-tab", "Q", ""},
		{"shift alone is allowed, with a warning", "close-tab", "Shift+Q", ""},
		{"a plain function key is fine", "close-tab", "F6", ""},
		{"Delete alone is fine", "close-tab", "Delete", "already"},
		{"Escape is reserved", "close-tab", "Ctrl+Escape", ""},
		{"Alt with a letter is fine", "close-tab", "Alt+Q", ""},
	}
	for _, tt := range tests {
		err := a.checkBinding(tt.id, mk(tt.combo))
		switch {
		case tt.name == "Delete alone is fine":
			// Delete is the default of "delete-row": taken, so it names that action
			if err == nil || !strings.Contains(err.Error(), "Delete row") {
				t.Errorf("%s: %v", tt.name, err)
			}
		case tt.name == "Escape is reserved":
			// not an error for Ctrl+Escape; plain Escape is tested below
		case tt.wantErr == "" && err != nil:
			t.Errorf("%s: %q refused: %v", tt.name, tt.combo, err)
		case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
			t.Errorf("%s: %q gave %v, want an error mentioning %q", tt.name, tt.combo, err, tt.wantErr)
		}
	}
	if err := a.checkBinding("close-tab", mk("Escape")); err == nil {
		t.Error("plain Escape belongs to closing overlays")
	}
}

func TestBindingWarningNamesPlainKeysOnly(t *testing.T) {
	tests := []struct {
		combo string
		warn  bool
	}{
		{"Q", true},
		{"Shift+Q", true},
		{"Ctrl+Q", false},
		{"Alt+Q", false},
		{"F6", false},
		{"Delete", false},
	}
	for _, tt := range tests {
		b, err := engine.ParseBinding(tt.combo)
		if err != nil {
			t.Fatal(err)
		}
		got := bindingWarning(b)
		if (got != "") != tt.warn || (tt.warn && !strings.Contains(got, "no text field")) {
			t.Errorf("bindingWarning(%s) = %q, want a warning: %v", tt.combo, got, tt.warn)
		}
	}
}

func TestAPlainKeyBindingFiresOnlyWhileNoTextFieldHasFocus(t *testing.T) {
	a, _ := newTestApp(t)
	a.settings.Shortcuts = map[string]string{"close-tab": "Q"}
	a.newRequest() // closing the only tab opens a fresh one, which would hide the effect
	tabs := len(a.tabs)
	a.cur().url.Focused = true
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyQ}})
	if len(a.tabs) != tabs {
		t.Fatal("typing Q in the URL field must not close the tab")
	}
	a.cur().url.Focused = false
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyQ}})
	if len(a.tabs) == tabs {
		t.Fatal("Q with no text field focused closes the tab")
	}
}

func TestTextKeysListEveryReservedCombination(t *testing.T) {
	var listed []string
	for _, row := range textKeys {
		listed = append(listed, row.keys)
	}
	all := strings.Join(listed, " ")
	for _, combo := range reservedForText {
		if !strings.Contains(all, combo) {
			t.Errorf("%s is reserved for text but not listed in the settings: %v", combo, listed)
		}
	}
}

// checkBinding says why combination b cannot be given to action id as it
// stands: refused outright, or taken by another action (which the editor
// offers to replace).
func (a *App) checkBinding(id string, b engine.Binding) error {
	if err := refuseBinding(b); err != nil {
		return err
	}
	if other, taken := a.conflictWith(id, b); taken {
		return fmt.Errorf("Already used by %s", other.name)
	}
	return nil
}
