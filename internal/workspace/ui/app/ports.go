package ui

import (
	"context"
	"errors"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fontfamily"
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

// noFonts is the font store of an app started without one: no families.
type noFonts struct{}

func (noFonts) Scan() []fontfamily.Family { return nil }
func (noFonts) Load(fontfamily.Family) ([]byte, []byte, error) {
	return nil, nil, errors.New("no font store")
}
