package respview

import (
	"bytes"
	"testing"
)

func TestLooksBinary(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want bool
	}{
		{"empty", nil, false},
		{"json", []byte(`{"a":"żółw"}`), false},
		{"nul byte", []byte("abc\x00def"), true},
		{"invalid utf-8", []byte{0xff, 0xfe, 'a'}, true},
		{"png header", []byte("\x89PNG\r\n\x1a\n\x00\x00"), true},
		{"a multi-byte character cut by the 8 KiB sample is still text", append(bytes.Repeat([]byte("a"), 8191), []byte("ż")...), false},
		{"binary after the sample is not seen", append(bytes.Repeat([]byte("a"), 9000), 0), false},
	}
	for _, tt := range tests {
		if got := LooksBinary(tt.in); got != tt.want {
			t.Errorf("%s: looksBinary = %v, want %v", tt.name, got, tt.want)
		}
	}
}
