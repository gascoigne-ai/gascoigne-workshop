package widget

import (
	"go/ast"
	"go/parser"
	"go/token"
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
