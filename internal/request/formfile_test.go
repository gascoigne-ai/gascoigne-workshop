package request

import "testing"

func TestFormFileReadsAtPathAsAFile(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		wantPath string
		wantOK   bool
	}{
		{"file", "@a.txt", "a.txt", true},
		{"path with a space", "@/home/u/my pic.png", "/home/u/my pic.png", true},
		{"windows drive path", `@C:\Users\u\a.txt`, `C:\Users\u\a.txt`, true},
		{"bare at is text", "@", "", false},
		{"double at is text", "@@x", "", false},
		{"no at is text", "x", "", false},
		{"empty is text", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := FormFile(tt.value)
			if path != tt.wantPath || ok != tt.wantOK {
				t.Fatalf("FormFile(%q) = %q, %v; want %q, %v", tt.value, path, ok, tt.wantPath, tt.wantOK)
			}
		})
	}
}

func TestAccessesTreatsStdioPrefixURLAsACommand(t *testing.T) {
	req := Request{Protocol: "mcp", URL: StdioPrefix + "cmd"}
	got := req.Accesses("")
	if len(got) != 1 || got[0] != (Access{AccessCommand, "cmd"}) {
		t.Fatalf("got %+v, want one command access for cmd", got)
	}
}
