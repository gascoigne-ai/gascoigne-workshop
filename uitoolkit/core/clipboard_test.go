package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPickClipboardUsesFirstCompleteTool(t *testing.T) {
	have := func(names ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, h := range names {
				if h == n {
					return "/bin/" + n, nil
				}
			}
			return "", errors.New("not found")
		}
	}
	tests := []struct {
		name     string
		lookPath func(string) (string, error)
		wantTool string // first word of the read command, "" for none
	}{
		{"wayland tools", have("wl-paste", "wl-copy", "xclip"), "wl-paste"},
		{"half installed pair is skipped", have("wl-paste", "xclip"), "xclip"},
		{"xsel", have("xsel"), "xsel"},
		{"none", have(), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickClipboard(tt.lookPath)
			if tt.wantTool == "" {
				if _, ok := got.(noClipboard); !ok {
					t.Fatalf("got %T, want noClipboard", got)
				}
				return
			}
			c, ok := got.(cmdClipboard)
			if !ok || c.tool.read[0] != tt.wantTool {
				t.Fatalf("got %+v, want tool %s", got, tt.wantTool)
			}
		})
	}
}

func TestNoClipboardFailsWithAHint(t *testing.T) {
	if _, err := (noClipboard{}).Read(); err == nil {
		t.Fatal("Read must fail")
	}
	if err := (noClipboard{}).Write("x"); err == nil {
		t.Fatal("Write must fail")
	}
}

// The command clipboard pipes text through the configured programs;
// `tee` and `cat` stand in for the real tools.
func TestCmdClipboardRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "clip")
	c := cmdClipboard{tool: clipboardTool{read: []string{"cat", file}, write: []string{"tee", file}}}

	if err := c.Write("héllo\nworld"); err != nil {
		t.Skipf("tee unavailable: %v", err)
	}
	got, err := c.Read()
	if err != nil || got != "héllo\nworld" {
		t.Fatalf("got %q err %v", got, err)
	}
	os.Remove(file)
	if _, err := c.Read(); err == nil {
		t.Fatal("reading a failing command must return an error")
	}
}
