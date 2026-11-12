package main

import "testing"

func TestParseWindowSize(t *testing.T) {
	tests := []struct {
		spec          string
		width, height int
		wantErr       bool
	}{
		{spec: "800x520", width: 800, height: 520},
		{spec: "1280x800", width: 1280, height: 800},
		{spec: "800", wantErr: true},
		{spec: "800x", wantErr: true},
		{spec: "axb", wantErr: true},
		{spec: "0x600", wantErr: true},
		{spec: "-5x600", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.spec, func(t *testing.T) {
			width, height, err := parseWindowSize(test.spec)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseWindowSize(%q) error = %v, want error %v", test.spec, err, test.wantErr)
			}
			if width != test.width || height != test.height {
				t.Errorf("parseWindowSize(%q) = %dx%d, want %dx%d", test.spec, width, height, test.width, test.height)
			}
		})
	}
}
