package ui

import (
	"fmt"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// statusClipboard reports clipboard results in the status bar: what was
// copied, or why the clipboard is not usable (no tool installed).
type statusClipboard struct {
	inner  widget.Clipboard
	report func(msg string, isErr bool)
}

func (s statusClipboard) Read() (string, error) {
	text, err := s.inner.Read()
	if err != nil {
		s.report(err.Error(), true)
	}
	return text, err
}

func (s statusClipboard) Write(text string) error {
	if err := s.inner.Write(text); err != nil {
		s.report(err.Error(), true)
		return err
	}
	s.report(fmt.Sprintf("Copied %d characters", utf8.RuneCountInString(text)), false)
	return nil
}
