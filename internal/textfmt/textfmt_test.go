package textfmt

import (
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
