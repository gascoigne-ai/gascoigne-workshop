package ui

import (
	"context"
	"errors"
)

// noFileDialog is the file chooser of an app started without one: callers
// ask for the path as text.
type noFileDialog struct{}

func (noFileDialog) Available() bool { return false }
func (noFileDialog) Choose(context.Context, string, bool, string) (string, error) {
	return "", errors.New("no file dialog")
}

// noBrowser is the preview of an app started without a browser.
type noBrowser struct{}

func (noBrowser) Show([]byte, string) error { return errors.New("no browser to open the preview in") }
