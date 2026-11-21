package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// sizeAxioms are the size constants that hold a bare number on purpose:
// the three bases and the fixed geometry style.md lists.
var sizeAxioms = map[string]bool{
	"BaseLine": true, "BaseSmallLine": true, "Unit": true,
	"DialogMargin": true, "DialogMaxH": true,
}

// Each size keeps its value from before the rules (the golden scenes depend
// on it) and equals its rule: a height is a base line plus padding on both
// sides, a spacing is a whole number of Units, a bar is a class plus a step.
func TestSizeConstantsFollowTheRules(t *testing.T) {
	tests := []struct {
		name      string
		got, rule int
		want      int
		units     int // the step in Units, for spacings; 0 for the rest
	}{
		{"RowH", RowH, BaseLine + 2*(3*Unit), 28, 0},
		{"CompactH", CompactH, BaseLine + 2*(2*Unit), 24, 0},
		{"SlimH", SlimH, BaseLine + 2*(1*Unit), 20, 0},
		{"LabelH", LabelH, BaseLine + 3*Unit, 22, 0},
		{"NoteH", NoteH, BaseSmallLine + 2*(1*Unit), 18, 0},
		{"TabLead", TabLead, StepHair, 2, 1},
		{"EdgeW", EdgeW, StepHair, 2, 1},
		{"NearGap", NearGap, StepNear, 4, 2},
		{"ShadowOffset", ShadowOffset, StepNear, 4, 2},
		{"TightGap", TightGap, StepTight, 6, 3},
		{"TabGap", TabGap, StepTight, 6, 3},
		{"RuleInset", RuleInset, StepTight, 6, 3},
		{"EdgeGrab", EdgeGrab, StepTight, 6, 3},
		{"LooseGap", LooseGap, StepLoose, 8, 4},
		{"CellPad", CellPad, StepLoose, 8, 4},
		{"LabelPad", LabelPad, StepLabel, 10, 5},
		{"TabSepInset", TabSepInset, StepLabel, 10, 5},
		{"Gutter", Gutter, StepGutter, 12, 6},
		{"DialogPad", DialogPad, StepDialog, 16, 8},
		{"ToolbarH", ToolbarH, RowH + 2*NearGap, 36, 0},
		{"DialogToolbarH", DialogToolbarH, RowH + 2*TightGap, 40, 0},
		{"TabBarH", TabBarH, RowH + 2*TightGap, 40, 0},
		{"DialogTitleH", DialogTitleH, RowH + 2*LooseGap, 44, 0},
		{"IconBtnW", IconBtnW, CompactH + 2*NearGap, 32, 0},
		{"WordBtnW", WordBtnW, IconBtnW + Gutter, 44, 0},
		{"CheckboxSize", CheckboxSize, BaseSmallLine, 14, 0},
		{"DialogMargin", DialogMargin, 40, 40, 0},
		{"DialogMaxH", DialogMaxH, 560, 560, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("%s = %d, want %d as before the rules", test.name, test.got, test.want)
			}
			if test.got != test.rule {
				t.Errorf("%s = %d, its rule gives %d", test.name, test.got, test.rule)
			}
			if test.units > 0 && test.got != test.units*Unit {
				t.Errorf("%s = %d, not step %d of Unit %d", test.name, test.got, test.units, Unit)
			}
		})
	}
}

// Every exported size constant in theme.go is a rule over the bases and
// steps, or one of the documented fixed numbers: a new bare number would
// bring back the free-floating sizes the rules replaced.
func TestEverySizeConstantIsARuleOrDocumentedFixed(t *testing.T) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "theme.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			if typeName, ok := value.Type.(*ast.Ident); ok && typeName.Name == "Token" {
				continue // color tokens, not sizes
			}
			for i, name := range value.Names {
				if !name.IsExported() || sizeAxioms[name.Name] || i >= len(value.Values) {
					continue
				}
				if !isRule(value.Values[i]) {
					t.Errorf("%s holds a free number: write it as a rule over the bases and steps, or list it as fixed in style.md and sizeAxioms", name.Name)
				}
			}
		}
	}
}

// isRule reports whether an expression is built from other constants, with
// numbers only as step counts (the 6 in BaseLine + 6*Unit). A number added
// on its own (RowH + 8) is a free offset, not a rule.
func isRule(expr ast.Expr) bool {
	named, freeNumber := false, false
	var walk func(node ast.Expr, counted bool)
	walk = func(node ast.Expr, counted bool) {
		switch node := node.(type) {
		case *ast.Ident:
			named = true
		case *ast.BasicLit:
			freeNumber = freeNumber || !counted
		case *ast.ParenExpr:
			walk(node.X, counted)
		case *ast.BinaryExpr:
			isCount := node.Op == token.MUL
			walk(node.X, isCount)
			walk(node.Y, isCount)
		default:
			freeNumber = true
		}
	}
	walk(expr, false)
	return named && !freeNumber
}
