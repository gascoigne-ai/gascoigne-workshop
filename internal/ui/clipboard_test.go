package ui

import (
	"errors"
	"testing"
)

type fakeClip struct {
	text string
	err  error
}

func (f *fakeClip) Read() (string, error) { return f.text, f.err }
func (f *fakeClip) Write(s string) error {
	if f.err != nil {
		return f.err
	}
	f.text = s
	return nil
}

func TestStatusClipboardReportsOutcome(t *testing.T) {
	var status string
	inner := &fakeClip{}
	c := statusClipboard{inner: inner, report: func(s string) { status = s }}

	if err := c.Write("żółw"); err != nil || inner.text != "żółw" || status != "Copied 4 characters" {
		t.Fatalf("write: err %v text %q status %q", err, inner.text, status)
	}
	inner.err = errors.New("no clipboard tool found")
	if _, err := c.Read(); err == nil || status != "no clipboard tool found" {
		t.Fatalf("read failure: err %v status %q", err, status)
	}
	if err := c.Write("x"); err == nil || status != "no clipboard tool found" {
		t.Fatalf("write failure: err %v status %q", err, status)
	}
}
