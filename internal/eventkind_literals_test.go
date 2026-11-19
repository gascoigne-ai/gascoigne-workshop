package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// A session event's kind is a port.EventKind constant. A string literal in
// a Kind field of the code that produces or shows events would bring back
// the untyped kinds a server's event name could collide with.
func TestEventKindsAreNeverStringLiterals(t *testing.T) {
	roots := []string{"request/adapter/proto", "workspace/ui/app", "workspace/controller/tabs"}
	fset := token.NewFileSet()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				pair, ok := node.(*ast.KeyValueExpr)
				if !ok {
					return true
				}
				key, isIdent := pair.Key.(*ast.Ident)
				literal, isLiteral := pair.Value.(*ast.BasicLit)
				if isIdent && key.Name == "Kind" && isLiteral && literal.Kind == token.STRING {
					t.Errorf("%s: Kind is the string literal %s; use a port.Kind constant", fset.Position(pair.Pos()), literal.Value)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
