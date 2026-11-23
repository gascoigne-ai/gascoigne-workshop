package keymap

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// parseTest reads "Ctrl+Shift+7" style text for the tests: modifiers, then a
// digit that is the key code.
func parseTest(text string) (Chord, bool, error) {
	if text == "" {
		return Chord{}, false, nil
	}
	var chord Chord
	parts := strings.Split(text, "+")
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "Ctrl":
			chord.Ctrl = true
		case "Shift":
			chord.Shift = true
		case "Alt":
			chord.Alt = true
		default:
			return Chord{}, false, errors.New("not a modifier")
		}
	}
	key := parts[len(parts)-1]
	if len(key) != 1 || key[0] < '0' || key[0] > '9' {
		return Chord{}, false, errors.New("not a key")
	}
	chord.Key = int(key[0] - '0')
	return chord, true, nil
}

func ctrl(key int) Chord { return Chord{Key: key, Ctrl: true} }

func commandsFor(ran *[]ID) []Command {
	run := func(id ID) func() { return func() { *ran = append(*ran, id) } }
	return []Command{
		{ID: "close", Name: "Close", Default: ctrl(1), Run: run("close")},
		{ID: "open", Name: "Open", Default: ctrl(2), Run: run("open")},
		{ID: "plain", Name: "Plain", Default: Chord{Key: 3}, Run: run("plain")},
	}
}

func TestResolveLayersValidOverridesOverDefaults(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		wantChord Chord
		wantBound bool
		wantUser  bool
	}{
		{"no override keeps the default", nil, ctrl(1), true, false},
		{"a valid override replaces it", map[string]string{"close": "Ctrl+7"}, ctrl(7), true, true},
		{"an empty string unbinds", map[string]string{"close": ""}, Chord{}, false, true},
		{"an unparsable value falls back to the default", map[string]string{"close": "Banana"}, ctrl(1), true, false},
		{"another command's override does not matter", map[string]string{"open": "Ctrl+7"}, ctrl(1), true, false},
		{"an unknown id is ignored", map[string]string{"teleport": "Ctrl+7"}, ctrl(1), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := Resolve(tt.overrides, commandsFor(new([]ID)), parseTest)
			chord, bound := keys.Chord("close")
			if bound != tt.wantBound || (bound && chord != tt.wantChord) {
				t.Errorf("Chord(close) = %+v, %v; want %+v, %v", chord, bound, tt.wantChord, tt.wantBound)
			}
			if got := keys.Overridden("close"); got != tt.wantUser {
				t.Errorf("Overridden(close) = %v, want %v", got, tt.wantUser)
			}
		})
	}
}

func TestResolveKeepsDisplayOrder(t *testing.T) {
	keys := Resolve(map[string]string{"close": ""}, commandsFor(new([]ID)), parseTest)
	var got []ID
	for _, command := range keys.Commands() {
		got = append(got, command.ID)
	}
	if want := []ID{"close", "open", "plain"}; !slices.Equal(got, want) {
		t.Fatalf("Commands() = %v, want %v even with an unbound command", got, want)
	}
}

func TestConflictNamesTheFirstOtherCommandOnTheChord(t *testing.T) {
	keys := Resolve(map[string]string{"plain": "Ctrl+1"}, commandsFor(new([]ID)), parseTest)
	if other, ok := keys.Conflict("open", ctrl(1)); !ok || other.ID != "close" {
		t.Errorf("Conflict(open, Ctrl+1) = %v, %v; want close, the first on the chord", other.ID, ok)
	}
	if _, ok := keys.Conflict("close", ctrl(2)); !ok {
		t.Error("Ctrl+2 is open's, so it conflicts for close")
	}
	if other, ok := keys.Conflict("open", ctrl(2)); ok {
		t.Errorf("a command never conflicts with itself, got %v", other.ID)
	}
	if _, ok := keys.Conflict("close", ctrl(9)); ok {
		t.Error("a free chord does not conflict")
	}
}

func TestRunRunsEveryCommandOnTheChordInOrder(t *testing.T) {
	var ran []ID
	keys := Resolve(map[string]string{"plain": "Ctrl+1"}, commandsFor(&ran), parseTest)
	if !keys.Run(Press{Chord: ctrl(1)}, false) {
		t.Fatal("Run reports false for a chord two commands have")
	}
	if want := []ID{"close", "plain"}; !slices.Equal(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	ran = nil
	if keys.Run(Press{Chord: Chord{Key: 1, Ctrl: true, Shift: true}}, false) || keys.Run(Press{Chord: Chord{Key: 9}}, false) {
		t.Error("Run reports true for a chord no command has, modifiers must match exactly")
	}
	if len(ran) != 0 {
		t.Errorf("ran %v on unbound chords", ran)
	}
}

func TestRunHoldsBackTypingKeysWhileTextIsFocused(t *testing.T) {
	tests := []struct {
		name        string
		press       Press
		textFocused bool
		want        bool
	}{
		{"a typing key with no text focused runs", Press{Chord: Chord{Key: 3}, TypesText: true}, false, true},
		{"a typing key while text is focused is for typing", Press{Chord: Chord{Key: 3}, TypesText: true}, true, false},
		{"a key that is not typed runs while text is focused", Press{Chord: ctrl(1)}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ran []ID
			keys := Resolve(nil, commandsFor(&ran), parseTest)
			if got := keys.Run(tt.press, tt.textFocused); got != tt.want || (len(ran) > 0) != tt.want {
				t.Errorf("Run = %v, ran %v; want %v", got, ran, tt.want)
			}
		})
	}
}

func TestAnUnboundCommandNeverRuns(t *testing.T) {
	var ran []ID
	commands := []Command{{ID: "zero", Name: "Zero", Default: Chord{Key: 0}, Run: func() { ran = append(ran, "zero") }}}
	keys := Resolve(map[string]string{"zero": ""}, commands, parseTest)
	if keys.Run(Press{Chord: Chord{}}, false) || len(ran) != 0 {
		t.Fatalf("an unbound command ran on key code 0: %v", ran)
	}
	if _, ok := keys.Conflict("other", Chord{}); ok {
		t.Fatal("an unbound command is not a conflict")
	}
	bound := Resolve(nil, commands, parseTest)
	if !bound.Run(Press{Chord: Chord{}}, false) {
		t.Fatal("key code 0 is a real key: the default binding must run")
	}
}
