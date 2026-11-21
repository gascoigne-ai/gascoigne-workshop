package widget

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// sizeOptions are the view options that take a size, padding or gap.
var sizeOptions = map[string]bool{
	"W": true, "H": true, "Gap": true, "Pad": true, "PadX": true, "PadY": true,
	"PadL": true, "PadR": true, "PadT": true, "PadB": true, "Separator": true,
}

// Tabs and menus take their sizes from the theme's size constants: no size
// option gets a raw number. Gap(0), the menu's "no look" placeholder, is
// not a size.
func TestTabAndMenuSizesComeFromThemeConstants(t *testing.T) {
	for _, file := range []string{"tabs.go", "menu.go"} {
		t.Run(file, func(t *testing.T) {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(parsed, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				option, ok := call.Fun.(*ast.Ident)
				if !ok || !sizeOptions[option.Name] {
					return true
				}
				for _, argument := range call.Args {
					literal, ok := argument.(*ast.BasicLit)
					if ok && literal.Kind == token.INT && literal.Value != "0" {
						t.Errorf("%s(%s) is a raw size", option.Name, literal.Value)
					}
				}
				return true
			})
		})
	}
}

// heightConstants are the default-face values of the size classes. A
// widget that reads one draws a row the same height whatever the text
// size is; it reads Ctx.Sizes instead (view.HOf, view.Sized, ctx.Sizes.Row).
var heightConstants = map[string]bool{
	"RowH": true, "CompactH": true, "SlimH": true, "LabelH": true, "NoteH": true,
	"ToolbarH": true, "DialogToolbarH": true, "TabBarH": true, "DialogTitleH": true,
	"IconBtnW": true, "WordBtnW": true, "CheckboxSize": true,
}

// keptConstants are the defining lines of exported constants that outside code (internal/**) still
// reads, so the toolkit keeps them as the default-face value. Each has a
// Ctx-based function beside it; a later task migrates the callers and
// removes the constant.
var keptConstants = map[string]bool{"menu.go:CompactH": true}

// No file under uitoolkit/ except the definitions (core/theme.go,
// core/sizes.go) and tests names a height constant: not as core.RowH and,
// inside package core, not as a bare RowH. The rule is on the syntax tree
// (a selector on the core import, or an identifier in package core), so a
// mention in a comment or in a longer name does not trip it.
func TestToolkitReadsHeightsFromCtxSizes(t *testing.T) {
	root := filepath.Join("..")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		slashed := filepath.ToSlash(path)
		if strings.HasSuffix(slashed, "core/theme.go") || strings.HasSuffix(slashed, "core/sizes.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		name := func(node ast.Node) string {
			switch node := node.(type) {
			case *ast.SelectorExpr:
				if pkg, ok := node.X.(*ast.Ident); ok && pkg.Name == "core" && heightConstants[node.Sel.Name] {
					return node.Sel.Name
				}
			case *ast.Ident:
				if parsed.Name.Name == "core" && heightConstants[node.Name] {
					return node.Name
				}
			}
			return ""
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			found := name(node)
			if found == "" || keptConstants[filepath.Base(path)+":"+found] {
				return true
			}
			t.Errorf("%s uses %s: read it from Ctx.Sizes", slashed, found)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
