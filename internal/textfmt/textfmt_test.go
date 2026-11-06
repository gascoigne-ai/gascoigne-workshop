package textfmt

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0ms"}, {850 * time.Millisecond, "850ms"}, {999 * time.Millisecond, "999ms"},
		{time.Second, "1.0s"}, {1400 * time.Millisecond, "1.4s"}, {61 * time.Second, "61.0s"},
	}
	for _, tt := range tests {
		if got := Duration(tt.in); got != tt.want {
			t.Errorf("Duration(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0B"}, {1023, "1023B"}, {1024, "1.00KB"}, {160430, "156.67KB"}, {3 << 20, "3.00MB"},
	}
	for _, tt := range tests {
		if got := Size(tt.in); got != tt.want {
			t.Errorf("Size(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCompactJSON(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantErr        string // substring of the error; "" means none
	}{
		{"spaces and lines go", "{\n  \"a\": 1,\n  \"b\": [1, 2]\n}", `{"a":1,"b":[1,2]}`, ""},
		{"key order and number text stay", `{"z": 1.50, "a": 12345678901234567890}`, `{"z":1.50,"a":12345678901234567890}`, ""},
		{"spaces inside strings stay", `{"a": "x  y"}`, `{"a":"x  y"}`, ""},
		{"surrounding blanks go", "  [1]  ", `[1]`, ""},
		{"a bare scalar is JSON too", ` "hi" `, `"hi"`, ""},
		{"broken JSON names where", `{"a": 1,, }`, "", "position 9"},
		{"empty text", "  ", "", "empty"},
		{"two values are not one message", `{"a":1} {"b":2}`, "", "more than one"},
	}
	for _, tt := range tests {
		got, err := CompactJSON(tt.in)
		switch {
		case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
			t.Errorf("%s: err = %v, want one mentioning %q", tt.name, err, tt.wantErr)
		case tt.wantErr == "" && (err != nil || got != tt.want):
			t.Errorf("%s: CompactJSON = %q, %v; want %q", tt.name, got, err, tt.want)
		}
	}
}
