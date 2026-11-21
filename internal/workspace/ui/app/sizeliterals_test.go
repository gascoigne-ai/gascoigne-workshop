package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"
)

// sizeAttribute finds a size, padding or gap attribute in a template whose
// value is a raw number instead of a constant read from the scope.
var sizeAttribute = regexp.MustCompile(`\s(w|h|gap|pad|pad-[xylrtb])="\d+"`)

// The status bar, file browser and welcome views take their sizes from the
// theme's size constants, so a template holds no raw numeric size and the
// browser's footer height is built from constants alone.
func TestViewSizesComeFromThemeConstants(t *testing.T) {
	for _, file := range []string{"statusbar.go", "filebrowser.go", "welcome.go"} {
		t.Run(file, func(t *testing.T) {
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, match := range sizeAttribute.FindAllString(string(source), -1) {
				t.Errorf("raw size attribute %s", strings.TrimSpace(match))
			}
		})
	}
	t.Run("browserFootH", func(t *testing.T) {
		for _, literal := range constLiterals(t, "filebrowser.go", "browserFootH") {
			if literal != "2" {
				t.Errorf("browserFootH holds the raw size %s", literal)
			}
		}
	})
}

// constLiterals lists the number literals in the value of the named constant.
func constLiterals(t *testing.T, file, name string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var literals []string
	found := false
	ast.Inspect(parsed, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || spec.Names[0].Name != name {
			return true
		}
		found = true
		for _, value := range spec.Values {
			ast.Inspect(value, func(inner ast.Node) bool {
				if literal, ok := inner.(*ast.BasicLit); ok && literal.Kind == token.INT {
					literals = append(literals, literal.Value)
				}
				return true
			})
		}
		return false
	})
	if !found {
		t.Fatalf("%s: no constant %s", file, name)
	}
	return literals
}
