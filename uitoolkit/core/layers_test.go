package core

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The toolkit must stay usable without the application: core imports
// nothing from this module, view only the core package, widget core and
// view, markup those and widget, and no toolkit
// package imports the application (internal or cmd).
func TestToolkitImportsStayInTheirLayer(t *testing.T) {
	const module = "github.com/gascoigne-ai/gascoigne-workshop"
	core := module + "/uitoolkit/core"
	view := module + "/uitoolkit/view"
	widget := module + "/uitoolkit/widget"
	markup := module + "/uitoolkit/markup"
	theming := module + "/uitoolkit/theming"
	engine := module + "/uitoolkit/engine"
	// The examples are test files and a main package, so they are checked
	// as well: a separate package is what keeps them to the exported API.
	allowed := map[string]layer{
		"../core":           {},
		"../view":           {imports: []string{core}},
		"../engine":         {imports: []string{core}},
		"../widget":         {imports: []string{core, view}},
		"../markup":         {imports: []string{core, view, widget}},
		"../theming":        {imports: []string{core}},
		"../examples":       {imports: []string{core, view, widget, markup, theming, engine}, tests: true},
		"../examples/hello": {imports: []string{core, view, widget, markup, theming, engine}, tests: true},
	}
	for dir, layer := range allowed {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no files in %s: %v", dir, err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") && !layer.tests {
				continue
			}
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			f, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range f.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if strings.HasPrefix(path, module) && !contains(layer.imports, path) {
					t.Errorf("%s imports %s", file, path)
				}
			}
		}
	}
}

// layer is what a toolkit directory may import; tests says whether its
// _test.go files are held to the same list.
type layer struct {
	imports []string
	tests   bool
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
