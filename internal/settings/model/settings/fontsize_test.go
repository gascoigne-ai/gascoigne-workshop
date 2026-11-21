package settings

import (
	"errors"
	"strings"
	"testing"
)

func TestClampFontSize(t *testing.T) {
	tests := []struct {
		name      string
		size      int
		want      int
		wantNotes int
	}{
		{"inside", 16, 16, 0},
		{"lowest", MinFontSize, MinFontSize, 0},
		{"highest", MaxFontSize, MaxFontSize, 0},
		{"zero is the default", 0, DefaultFontSize, 1},
		{"below", 9, MinFontSize, 1},
		{"negative", -5, MinFontSize, 1},
		{"above", 99, MaxFontSize, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prefs := Default()
			prefs.FontSize = test.size
			clamped, notes := Clamp(prefs)
			if clamped.FontSize != test.want || len(notes) != test.wantNotes {
				t.Fatalf("Clamp(%d) = %d with notes %q; want %d with %d notes", test.size, clamped.FontSize, notes, test.want, test.wantNotes)
			}
			if test.wantNotes > 0 && !strings.Contains(notes[0], "fontSize") {
				t.Fatalf("note %q does not name fontSize", notes[0])
			}
		})
	}
}

func TestFontSizeRow(t *testing.T) {
	field, ok := FieldFor("fontSize")
	if !ok || field.Kind != Int || field.Min != 10 || field.Max != 24 || field.Title != "Text size" {
		t.Fatalf("FieldFor(fontSize) = %+v, %v", field, ok)
	}
	if got := Expectation(field); got != "a whole number from 10 to 24" {
		t.Fatalf("Expectation = %q", got)
	}
	if _, known := FieldFor("textSize"); known {
		t.Fatal("the legacy key is in the schema")
	}
}

func TestValidateFontSize(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantMessage string
	}{
		{"inside", `{"version": 2, "appearance": {"fontSize": 16}}`, ""},
		{"lowest", `{"version": 2, "appearance": {"fontSize": 10}}`, ""},
		{"below", `{"version": 2, "appearance": {"fontSize": 9}}`, "must be a whole number between 10 and 24"},
		{"above", `{"version": 2, "appearance": {"fontSize": 25}}`, "must be a whole number between 10 and 24"},
		{"fractional", `{"version": 2, "appearance": {"fontSize": 14.5}}`, "must be a whole number between 10 and 24"},
		{"text", `{"version": 2, "appearance": {"fontSize": "large"}}`, "must be a number"},
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
