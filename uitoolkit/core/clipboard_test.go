package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPickClipboardUsesFirstCompleteTool(t *testing.T) {
	have := func(names ...string) func(string) (string, error) {
		return func(name string) (string, error) {
			for _, h := range names {
				if h == name {
					return "/bin/" + name, nil
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

type fakeClipboard struct {
	text string
	err  error
}

func (f *fakeClipboard) Read() (string, error) { return f.text, f.err }
func (f *fakeClipboard) Write(text string) error {
	if f.err != nil {
		return f.err
	}
	f.text = text
	return nil
}

func TestReportClipboardReportsOutcome(t *testing.T) {
	var message string
	var isErr bool
	inner := &fakeClipboard{}
	clipboard := ReportClipboard(inner, func(text string, failed bool) { message, isErr = text, failed })

	if err := clipboard.Write("żółw"); err != nil || inner.text != "żółw" || message != "Copied 4 characters" || isErr {
		t.Fatalf("write: err %v text %q message %q isErr %v", err, inner.text, message, isErr)
	}
	inner.err = errors.New("no clipboard tool found")
	if _, err := clipboard.Read(); err == nil || message != "no clipboard tool found" || !isErr {
		t.Fatalf("read failure: err %v message %q isErr %v", err, message, isErr)
	}
	if err := clipboard.Write("x"); err == nil || message != "no clipboard tool found" || !isErr {
		t.Fatalf("write failure: err %v message %q isErr %v", err, message, isErr)
	}
}

func TestReportClipboardStaysQuietOnASuccessfulRead(t *testing.T) {
	reported := false
	clipboard := ReportClipboard(&fakeClipboard{text: "pasted"}, func(string, bool) { reported = true })
	if text, err := clipboard.Read(); text != "pasted" || err != nil || reported {
		t.Fatalf("read: %q, %v, reported %v", text, err, reported)
	}
}

// A clipboard of hundreds of MB was read whole into one string and then
// into every undo snapshot. Text over MaxPasteBytes is refused with a
// message that names the limit.
func TestReadRefusesClipboardTextOverTheLimit(t *testing.T) {
	file := filepath.Join(t.TempDir(), "clip")
	if err := os.WriteFile(file, make([]byte, MaxPasteBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	c := cmdClipboard{tool: clipboardTool{read: []string{"cat", file}, write: []string{"tee", file}}}
	text, err := c.Read()
	if err == nil || text != "" || !strings.Contains(err.Error(), "1 MiB") {
		t.Fatalf("got %d bytes, err %v; want a refusal naming 1 MiB", len(text), err)
	}
	if err := os.WriteFile(file, make([]byte, MaxPasteBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	if text, err := c.Read(); err != nil || len(text) != MaxPasteBytes {
		t.Fatalf("text of exactly the limit: %d bytes, err %v", len(text), err)
	}
}
