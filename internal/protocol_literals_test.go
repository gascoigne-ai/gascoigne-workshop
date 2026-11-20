package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// protocolLiteralFiles may name a protocol, each for a reason. Every other
// file reaches what differs per protocol through its row of
// model.Protocols or of the app's protocolViews (see the Protocols doc).
// Paths are
// relative to internal/.
var protocolLiteralFiles = map[string]string{
	"request/model/protocol.go":                      "the rows of model.Protocols and the parser",
	"request/adapter/proto/proto.go":                 "the registry: one implementation per row",
	"settings/model/settings/schema.go":              "settings keys that switch the experimental protocols on",
	"request/adapter/proto/ws/ws.go":                 "the HTTP Upgrade token of RFC 6455",
	"request/adapter/exampleserver/exampleserver.go": "names of the example server's services",
	"../cmd/example-server/main.go":                  "command line flags of the example server",
}

// protocolConstants are the names of the model.Protocol constants.
var protocolConstants = map[string]bool{
	"ProtocolREST": true, "ProtocolWebSocket": true, "ProtocolMCP": true, "ProtocolSSE": true, "ProtocolGRPC": true,
}

// protocolComparisons lists the places in file that name a protocol: a
// string literal that is a protocol's name, and a comparison, case or map
// key against a protocol constant. Building a value from a constant
// (Protocol: model.ProtocolREST) is not one.
func protocolComparisons(fset *token.FileSet, file *ast.File) []string {
	var found []string
	report := func(node ast.Node, what string) {
		found = append(found, fmt.Sprintf("%s: %s", fset.Position(node.Pos()), what))
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BasicLit:
			if text, err := strconv.Unquote(node.Value); err == nil && node.Kind == token.STRING && isProtocolName(text) {
				report(node, "protocol literal "+node.Value)
			}
		case *ast.BinaryExpr:
			if (node.Op == token.EQL || node.Op == token.NEQ) && (isProtocolConstant(node.X) || isProtocolConstant(node.Y)) {
				report(node, "comparison with a protocol")
			}
		case *ast.CaseClause:
			if slices.ContainsFunc(node.List, isProtocolConstant) {
				report(node, "case on a protocol")
			}
		case *ast.KeyValueExpr:
			if isProtocolConstant(node.Key) {
				report(node, "map keyed by a protocol")
			}
		}
		return true
	})
	return found
}

func isProtocolName(text string) bool {
	for _, row := range model.Protocols {
		if text == string(row.Name) {
			return true
		}
	}
	return false
}

func isProtocolConstant(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.Ident:
		return protocolConstants[expr.Name]
	case *ast.SelectorExpr:
		return protocolConstants[expr.Sel.Name]
	}
	return false
}

func TestTheProtocolGuardCatchesEachKind(t *testing.T) {
	tests := []struct {
		name, source string
		want         int
	}{
		{"literal", `package p; var a = req.Protocol == "mcp"`, 1},
		{"comparison", `package p; var a = req.Protocol != model.ProtocolREST`, 1},
		{"case", `package p; func f() { switch p { case model.ProtocolSSE: } }`, 1},
		{"map key", `package p; var m = map[model.Protocol]bool{model.ProtocolGRPC: true}`, 1},
		{"construction is fine", `package p; var r = model.Request{Protocol: model.ProtocolREST}`, 0},
		{"other strings are fine", `package p; var s = "restful" + "Last-Event-ID"`, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", test.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := protocolComparisons(fset, file); len(got) != test.want {
				t.Fatalf("found %q, want %d", got, test.want)
			}
		})
	}
}

func TestNoProtocolComparisonOutsideTheTable(t *testing.T) {
	var found []string
	for _, root := range []string{".", "../cmd"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			if _, allowed := protocolLiteralFiles[filepath.ToSlash(path)]; allowed {
				return nil
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			found = append(found, protocolComparisons(fset, file)...)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(found) > 0 {
		t.Fatalf("per-protocol behavior belongs in a row of model.Protocols or protocolViews:\n%s", strings.Join(found, "\n"))
	}
}
