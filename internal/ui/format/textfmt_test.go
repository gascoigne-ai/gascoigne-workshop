package format

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

func TestJSONStatus(t *testing.T) {
	tests := []struct {
		name, in  string
		wantValid bool
		wantMsg   string // substring; "" means no message at all
	}{
		{"empty body has nothing to report", "", false, ""},
		{"whitespace only too", "  \n", false, ""},
		{"valid object", `{"a": [1, 2]}`, true, "Valid JSON"},
		{"valid scalar", `42`, true, "Valid JSON"},
		{"invalid", `{"a": }`, false, "Invalid JSON"},
		{"truncated", `{"a": 1`, false, "Invalid JSON"},
	}
	for _, test := range tests {
		msg, valid := JSONStatus(test.in)
		if valid != test.wantValid || (test.wantMsg == "") != (msg == "") || !strings.Contains(msg, test.wantMsg) {
			t.Errorf("%s: got %q %v; want %q %v", test.name, msg, valid, test.wantMsg, test.wantValid)
		}
	}
}

func TestIndentJSON(t *testing.T) {
	got, err := IndentJSON(`{"a":[1,2],"b":{}}`)
	want := "{\n  \"a\": [\n    1,\n    2\n  ],\n  \"b\": {}\n}"
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
	if _, err := IndentJSON(`{"a": }`); err == nil {
		t.Fatal("invalid JSON must be an error")
	}
}
