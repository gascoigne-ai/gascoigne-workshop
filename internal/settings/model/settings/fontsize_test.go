package settings

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateFontSize(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantMessage string
	}{
		{"inside", `{"version": 3, "appearance": {"fontSize": 16}}`, ""},
		{"lowest", `{"version": 3, "appearance": {"fontSize": 10}}`, ""},
		{"below", `{"version": 3, "appearance": {"fontSize": 9}}`, "must be a whole number between 10 and 24"},
		{"above", `{"version": 3, "appearance": {"fontSize": 25}}`, "must be a whole number between 10 and 24"},
		{"fractional", `{"version": 3, "appearance": {"fontSize": 14.5}}`, "must be a whole number between 10 and 24"},
		{"text", `{"version": 3, "appearance": {"fontSize": "large"}}`, "must be a number"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			problems := Validate([]byte(test.text))
			if test.wantMessage == "" && len(problems) == 0 {
				return
			}
			if len(problems) != 1 || problems[0].Key != "fontSize" || problems[0].Message != test.wantMessage {
				t.Fatalf("Validate(%s) = %+v, want %q on fontSize", test.text, problems, test.wantMessage)
			}
		})
	}
}

func TestSetFontSizeRefusesText(t *testing.T) {
	prefs := Default()
	if err := Set(&prefs, "fontSize", Text("large")); !errors.Is(err, ErrWrongValue) {
		t.Fatalf("Set(fontSize, Text) = %v, want ErrWrongValue", err)
	}
}

func TestMinimalWritesFontSizeOnlyWhenChanged(t *testing.T) {
	prefs := Default()
	text, err := Minimal(prefs)
	if err != nil || strings.Contains(string(text), "fontSize") {
		t.Fatalf("Minimal(default) = %s, %v; want no fontSize", text, err)
	}
	prefs.FontSize = 18
	text, err = Minimal(prefs)
	if err != nil || !strings.Contains(string(text), `"fontSize": 18`) {
		t.Fatalf("Minimal(18) = %s, %v; want fontSize 18", text, err)
	}
}
