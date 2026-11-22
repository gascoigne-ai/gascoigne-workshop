package app

import (
	"context"
	"errors"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

// noFileDialog is the file chooser of an app started without one: callers
// get the in-app file browser.
type noFileDialog struct{}

func (noFileDialog) Available() bool   { return false }
func (noFileDialog) Suggested() string { return "" }
func (noFileDialog) Choose(context.Context, string, workspace.FileMode, string) (string, error) {
	return "", errors.New("no file dialog")
}

// noFolders is the file system of an app started without one: the file
// browser shows no folder.
type noFolders struct{}

func (noFolders) List(string) ([]workspace.FolderEntry, error) {
	return nil, errors.New("no file system")
}
func (noFolders) Home() (string, error) { return "", errors.New("no file system") }
func (noFolders) Write(context.Context, string, []byte) error {
	return errors.New("no file system")
}
func (noFolders) IsFile(context.Context, string) bool { return false }

// noBrowser is the preview of an app started without a browser.
type noBrowser struct{}

func (noBrowser) Show(context.Context, []byte, string) error {
	return errors.New("no browser to open the preview in")
}

// noFonts is the font store of an app started without one: no families.
type noFonts struct{}

func (noFonts) Scan() []settings.Family { return nil }
func (noFonts) Load(settings.Family) ([]byte, []byte, error) {
	return nil, nil, errors.New("no font store")
}

// noStore is the collection store of an app started without one.
type noStore struct{}

func (noStore) Load(string) (request.Collection, error) {
	return request.Collection{}, errors.New("no collection store")
}
func (noStore) Save(string, request.Collection) error { return errors.New("no collection store") }

func (noStore) Create(string, string) (string, error) { return "", errors.New("no collection store") }
func (noStore) Rename(string, string) (string, error) { return "", errors.New("no collection store") }
func (noStore) Delete(string) error                   { return errors.New("no collection store") }
func (noStore) List(string) ([]string, error)         { return nil, errors.New("no collection store") }

// noSettingFiles is the settings store of an app started without one: it
// reads the defaults and cannot save.
type noSettingFiles struct{}

func (noSettingFiles) Load(string) (settings.Settings, error) { return settings.Default(), nil }
func (noSettingFiles) Save(string, settings.Settings) error {
	return errors.New("no settings store")
}
func (noSettingFiles) ReadText(string) ([]byte, error) { return nil, errors.New("no settings store") }
func (noSettingFiles) Stat(string) settings.FileStamp  { return settings.FileStamp{} }
func (noSettingFiles) WriteText(string, []byte) error  { return errors.New("no settings store") }

// noStates keeps no state between runs.
type noStates struct{}

func (noStates) Load(string) (state.State, error) { return state.State{}, nil }
func (noStates) Save(string, state.State) error   { return nil }
