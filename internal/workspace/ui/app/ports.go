package ui

import (
	"context"
	"errors"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fontfamily"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
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

// noStore is the collection store of an app started without one.
type noStore struct{}

func (noStore) Load(string) (model.Collection, error) {
	return model.Collection{}, errors.New("no collection store")
}
func (noStore) Save(string, model.Collection) error { return errors.New("no collection store") }

// noSettingFiles is the settings store of an app started without one: it
// reads the defaults and cannot save.
type noSettingFiles struct{}

func (noSettingFiles) Load(string) (settings.Settings, error) { return settings.Default(), nil }
func (noSettingFiles) Save(string, settings.Settings) error {
	return errors.New("no settings store")
}
func (noSettingFiles) WriteText(string, []byte) error { return errors.New("no settings store") }

// noStates keeps no state between runs.
type noStates struct{}

func (noStates) Load(string) (state.State, error) { return state.State{}, nil }
func (noStates) Save(string, state.State) error   { return nil }
