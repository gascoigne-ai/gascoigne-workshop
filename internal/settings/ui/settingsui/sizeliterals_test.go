package settingsui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sizeAttribute finds a size, padding, gap or margin attribute in a template
// whose value is a raw non-zero number instead of a name read from the scope.
var sizeAttribute = regexp.MustCompile(`\s((?:min-|max-)?[wh]|gap|pad|pad-[xylrtb]|margin|margin-[xylrtb])="([1-9]\d*)"`)

// sizeOptions are the view options whose argument is a size in pixels.
var sizeOptions = map[string]bool{
	"W": true, "H": true, "MinW": true, "MaxW": true, "MinH": true, "MaxH": true,
	"Gap": true, "Pad": true, "PadX": true, "PadY": true, "PadL": true, "PadT": true, "PadR": true, "PadB": true,
	"Margin": true, "MarginX": true, "MarginY": true, "MarginL": true, "MarginT": true, "MarginR": true, "MarginB": true,
}

// heightConstants are the default-face constants that ctx.Sizes replaces: a
// view reads the size of the face it draws with, not the default's.
var heightConstants = map[string]bool{
	"RowH": true, "CompactH": true, "SlimH": true, "LabelH": true, "NoteH": true, "ToolbarH": true,
	"DialogToolbarH": true, "TabBarH": true, "DialogTitleH": true, "IconBtnW": true, "WordBtnW": true,
	"CheckboxSize": true, "MenuRowH": true, "MessageDialogW": true,
}

// scopeSizeKey names the scope entries that hold a size.
var scopeSizeKey = regexp.MustCompile(`(H|W|Gap|Pad|Size)$`)

// sizeLiteralAllowed lists "file: finding" strings that may stay; it only
// shrinks. Test files keep the default-face constants because their
// coordinates are those of the default scene.
var sizeLiteralAllowed = map[string]bool{}

// featureViewDirs are the feature view packages the guard covers. They cannot
// share a helper package (a ui package imports no other ui package), so one
// test reads their files from disk.
var featureViewDirs = []string{
	"../../../collection/ui/*", "../../../request/ui/*", "../../../response/ui/*", "../../../settings/ui/*",
}

// A feature view takes its sizes from Ctx.Sizes and the named roles, so no
// file holds a raw size in a template or a size option, and none names a
// default-face height constant.
func TestFeatureViewSizesComeFromCtxSizes(t *testing.T) {
	var files []string
	for _, pattern := range featureViewDirs {
		found, err := filepath.Glob(pattern + "/*.go")
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	if len(files) == 0 {
		t.Fatal("no feature view files found")
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		t.Run(file, func(t *testing.T) {
			for _, finding := range sizeFindings(t, file) {
				if !sizeLiteralAllowed[file+": "+finding] {
					t.Error(finding)
				}
			}
		})
	}
}

// sizeFindings lists the raw sizes and old constants in one file.
func sizeFindings(t *testing.T, file string) []string {
	t.Helper()
	source, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var findings []string
	where := func(node ast.Node) string { return fmt.Sprintf("line %d", fset.Position(node.Pos()).Line) }
	test := strings.HasSuffix(file, "_test.go")
	if !test {
		for _, match := range sizeAttribute.FindAllStringSubmatch(string(source), -1) {
			findings = append(findings, fmt.Sprintf("raw size attribute %s=%q", match[1], match[2]))
		}
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.SelectorExpr:
			if heightConstants[node.Sel.Name] && (!test || node.Sel.Name == "MenuRowH" || node.Sel.Name == "MessageDialogW") {
				findings = append(findings, fmt.Sprintf("%s: old constant %s", where(node), node.Sel.Name))
			}
		case *ast.Ident: // a name dot-imported from the widget package
			if node.Name == "MenuRowH" || node.Name == "MessageDialogW" {
				findings = append(findings, fmt.Sprintf("%s: old constant %s", where(node), node.Name))
			}
		case *ast.CallExpr:
			name := calleeName(node)
			if !sizeOptions[name] || test {
				return true
			}
			for _, argument := range node.Args {
				if hasAddedLiteral(argument) {
					findings = append(findings, fmt.Sprintf("%s: raw size in %s()", where(node), name))
				}
			}
		case *ast.KeyValueExpr:
			key, ok := node.Key.(*ast.BasicLit)
			if !ok || key.Kind != token.STRING || test {
				return true
			}
			if scopeSizeKey.MatchString(strings.Trim(key.Value, `"`)) && hasAddedLiteral(node.Value) {
				findings = append(findings, fmt.Sprintf("%s: raw size in scope entry %s", where(node), key.Value))
			}
		}
		return true
	})
	return findings
}

// calleeName is the function name of a call, with its package dropped.
func calleeName(call *ast.CallExpr) string {
	switch function := call.Fun.(type) {
	case *ast.Ident:
		return function.Name
	case *ast.SelectorExpr:
		return function.Sel.Name
	}
	return ""
}

// hasAddedLiteral reports a non-zero number that stands alone or is added or
// subtracted; a factor (2*w.Gutter) is a rule over a constant, not a size.
func hasAddedLiteral(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		return expr.Kind == token.INT && expr.Value != "0"
	case *ast.ParenExpr:
		return hasAddedLiteral(expr.X)
	case *ast.BinaryExpr:
		if expr.Op == token.ADD || expr.Op == token.SUB {
			return hasAddedLiteral(expr.X) || hasAddedLiteral(expr.Y)
		}
	}
	return false
}
