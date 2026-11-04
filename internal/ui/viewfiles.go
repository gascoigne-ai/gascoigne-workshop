package ui

import (
	"embed"
	"io/fs"
	"os"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/component"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Views written as templates (uitoolkit/component) are embedded here; the
// -views flag reads them from a folder instead, and edits show without
// a restart.
//
//go:embed views/*.vml
var viewFiles embed.FS

// newViewLoader reads templates from dir when it is set, else from the
// embedded files.
func newViewLoader(dir string) *component.Loader {
	if dir != "" {
		return component.NewLoader(os.DirFS(dir))
	}
	sub, err := fs.Sub(viewFiles, "views")
	if err != nil {
		panic(err) // the embedded directory exists
	}
	return component.NewLoader(sub)
}

// templateView builds the template called name for scope. A template that
// cannot be built shows its error where the view would be, and the
// status line, so an edit in progress is visible and not a crash.
func (a *App) templateView(name string, scope component.Scope) Node {
	n, err := a.views.Build(name, scope)
	if err != nil {
		return Text(err.Error(), Color(w.Error), Small())
	}
	return n
}
