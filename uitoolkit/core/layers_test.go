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
// nothing from this module, vml only the core package, and no toolkit
// package imports the application (internal or cmd).
func TestToolkitImportsStayInTheirLayer(t *testing.T) {
	const module = "github.com/gascoigne-ai/gascoigne-workshop"
	allowed := map[string][]string{
		"../core":    {},
		"../vml":       {module + "/uitoolkit/core"},
		"../engine":    {module + "/uitoolkit/core"},
		"../component": {module + "/uitoolkit/core", module + "/uitoolkit/vml"},
		"../theming":   {module + "/uitoolkit/core"},
	}
	for dir, ok := range allowed {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no files in %s: %v", dir, err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
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
				if strings.HasPrefix(path, module) && !contains(ok, path) {
					t.Errorf("%s imports %s", file, path)
				}
			}
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
