package form

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

func TestUpdate(t *testing.T) {
	withTimeout := func(seconds int) settings.Settings {
		current := settings.Default()
		current.TimeoutSec = seconds
		return current
	}
	timeout, _ := settings.FieldFor("timeoutSec")
	refusal := fmt.Sprintf("Enter a whole number between %d and %d", timeout.Min, timeout.Max)
	refused := Model{notes: map[string]string{"timeoutSec": refusal}}

	tests := []struct {
		name        string
		model       Model
		current     settings.Settings
		msg         Msg
		wantNote    string
		wantEffects []Effect
	}{
		{"a number in range is applied", Model{}, withTimeout(60), Typed{"timeoutSec", " 20 "}, "",
			[]Effect{Apply{"timeoutSec", settings.Number(20)}}},
		{"a number below the range is refused with its range", Model{}, withTimeout(60), Typed{"timeoutSec", "0"}, refusal, nil},
		{"text that is no number is refused", Model{}, withTimeout(60), Typed{"timeoutSec", "soon"}, refusal, nil},
		{"a good value clears an earlier refusal", refused, withTimeout(60), Typed{"timeoutSec", "20"}, "",
			[]Effect{Apply{"timeoutSec", settings.Number(20)}}},
		{"a text setting is applied as typed, trimmed", Model{}, settings.Default(), Typed{"proxy", " http://p:1 "}, "",
			[]Effect{Apply{"proxy", settings.Text("http://p:1")}}},
		{"an unknown key does nothing", refused, settings.Default(), Typed{"nope", "1"}, refusal, nil},
		{"a step fills the field and applies the stepped number", Model{}, withTimeout(60), Stepped{"timeoutSec", 1}, "",
			[]Effect{Fill{"timeoutSec", "61"}, Apply{"timeoutSec", settings.Number(61)}}},
		{"a step starts from the setting, not from a refused entry", refused, withTimeout(60), Stepped{"timeoutSec", -1}, "",
			[]Effect{Fill{"timeoutSec", "59"}, Apply{"timeoutSec", settings.Number(59)}}},
		{"a step past the top does nothing", Model{}, withTimeout(timeout.Max), Stepped{"timeoutSec", 1}, "", nil},
		{"a step past the bottom does nothing", Model{}, withTimeout(timeout.Min), Stepped{"timeoutSec", -1}, "", nil},
		{"a reset restores the key and forgets notes and typed text", refused, withTimeout(20), Reset{"timeoutSec"}, "",
			[]Effect{Restore{"timeoutSec"}, ForgetTexts{}}},
		{"resetting a font asks for the fonts again", Model{}, settings.Default(), Reset{"uiFont"}, "",
			[]Effect{Restore{"uiFont"}, ForgetTexts{}, ApplyFonts{}}},
		{"choosing a file asks for it with the title of the setting", Model{}, settings.Default(), Choose{"caCert"}, "",
			[]Effect{AskFile{"caCert", "Choose CA certificates"}}},
		{"a picked file fills the field and is applied", Model{}, settings.Default(), Picked{"caCert", "/etc/ssl/ca.pem"}, "",
			[]Effect{Fill{"caCert", "/etc/ssl/ca.pem"}, Apply{"caCert", settings.Text("/etc/ssl/ca.pem")}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, effects := Update(test.model, test.current, test.msg)
			if got.Note("timeoutSec") != test.wantNote {
				t.Fatalf("note %q, want %q", got.Note("timeoutSec"), test.wantNote)
			}
			if !reflect.DeepEqual(effects, test.wantEffects) {
				t.Fatalf("effects %#v, want %#v", effects, test.wantEffects)
			}
		})
	}
}

func TestUpdateLeavesTheModelItWasGivenAlone(t *testing.T) {
	before := Model{notes: map[string]string{"timeoutSec": "x"}}
	Update(before, settings.Default(), Typed{"timeoutSec", "20"})
	if before.Note("timeoutSec") != "x" {
		t.Fatal("Update changed the model it was given")
	}
}

func TestTextOfShowsTheValueOfASetting(t *testing.T) {
	current := settings.Default()
	current.TimeoutSec = 20
	current.Proxy = "http://p:1"
	for key, want := range map[string]string{"timeoutSec": "20", "proxy": "http://p:1", "layout": "stacked", "nope": ""} {
		if got := TextOf(current, key); got != want {
			t.Errorf("TextOf(%s) = %q, want %q", key, got, want)
		}
	}
}
