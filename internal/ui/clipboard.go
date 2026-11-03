package ui

import (
	"fmt"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// statusClipboard reports clipboard results in the status bar: what was
// copied, or why the clipboard is not usable (no tool installed).
type statusClipboard struct {
	inner  widget.Clipboard
	report func(string)
}

func (s statusClipboard) Read() (string, error) {
	text, err := s.inner.Read()
	if err != nil {
		s.report(err.Error())
	}
	return text, err
}

func (s statusClipboard) Write(text string) error {
	if err := s.inner.Write(text); err != nil {
		s.report(err.Error())
		return err
	}
	s.report(fmt.Sprintf("Copied %d characters", utf8.RuneCountInString(text)))
	return nil
}
