package internal

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const modulePrefix = "github.com/gascoigne-ai/gascoigne-workshop/internal/"

// imports maps a package path under internal/ (like "request/adapter/proto")
// to the internal packages its non-test files import, in the same form.
type imports map[string][]string

// exceptions are known violations: the App in workspace/ui/app still reads
// and writes the other domains' adapters, controllers and views directly.
// Splitting it into a controller and a view behind ports removes them (task
// 033). Each entry is "importer -> imported". The list only shrinks.
var exceptions = map[string]bool{
	"workspace/ui/app -> collection/adapter/openapi":     true,
	"workspace/ui/app -> collection/adapter/postman":     true,
	"workspace/ui/app -> collection/adapter/store":       true,
	"workspace/ui/app -> request/adapter/proto":          true,
	"workspace/ui/app -> request/controller/assert":      true,
	"workspace/ui/app -> request/ui/calltrace":           true,
	"workspace/ui/app -> response/adapter/history":       true,
	"workspace/ui/app -> response/adapter/preview":       true,
	"workspace/ui/app -> response/ui/respview":           true,
	"workspace/ui/app -> response/ui/sentlog":            true,
	"workspace/ui/app -> settings/adapter/fonts":         true,
	"workspace/ui/app -> settings/controller/appearance": true,
	"workspace/ui/app -> settings/ui/settingsui":         true,
	"workspace/ui/app -> workspace/adapter/filedialog":   true,
	"workspace/ui/app -> workspace/adapter/state":        true,
}

// violations applies the layer rules to the packages that already sit in a
// domain: domain/role/..., where role is ui, controller, port, adapter or model.
// Packages still at the top of internal/ are not checked yet.
//
//   - model imports only other model packages;
//   - port imports only model and port;
//   - controller imports model, port and controller, never ui or adapter;
//   - ui and adapter import model, port and controller, never each other;
//   - another domain may be reached only through its model or port.
func violations(all imports, allowed map[string]bool) []string {
	var found []string
	for importer, targets := range all {
		from, ok := locate(importer)
		if !ok {
			continue
		}
		for _, target := range targets {
			to, ok := locate(target)
			if !ok || allowed[importer+" -> "+target] {
				continue
			}
			if reason := broken(from, to); reason != "" {
				found = append(found, importer+" -> "+target+": "+reason)
			}
		}
	}
	slices.Sort(found)
	return found
}

// place is where a package sits: its domain and its role.
type place struct{ domain, role string }

// canImport lists, per role, the roles it may import.
var canImport = map[string][]string{
	"model":      {"model"},
	"port":       {"model", "port"},
	"controller": {"model", "port", "controller"},
	"ui":         {"model", "port", "controller", "ui"},
	"adapter":    {"model", "port", "controller", "adapter"},
}

// outsideRoles are what another domain may be reached through.
var outsideRoles = []string{"model", "port"}

func locate(pkg string) (place, bool) {
	parts := strings.Split(pkg, "/")
	if len(parts) < 2 {
		return place{}, false
	}
	_, known := canImport[parts[1]]
	return place{parts[0], parts[1]}, known
}

func broken(from, to place) string {
	if from.domain != to.domain && !slices.Contains(outsideRoles, to.role) {
		return "a domain is reached only through its model or port, not its " + to.role
	}
	if !slices.Contains(canImport[from.role], to.role) {
		return from.role + " must not import " + to.role
	}
	return ""
}

func TestRulesCatchEachKindOfViolation(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		bad  bool
	}{
		{"model imports model", "request/model", "collection/model/collection", false},
		{"model imports adapter", "request/model", "request/adapter/proto", true},
		{"model imports ui", "request/model", "response/ui/respview", true},
		{"port imports model", "request/port", "request/model", false},
		{"port imports adapter", "request/port", "request/adapter/proto", true},
		{"controller imports port", "request/controller/assert", "request/port", false},
		{"controller imports adapter", "request/controller/assert", "request/adapter/proto", true},
		{"controller imports ui", "request/controller/assert", "request/ui/calltrace", true},
		{"ui imports controller", "request/ui/calltrace", "request/controller/assert", false},
		{"ui imports adapter", "request/ui/calltrace", "request/adapter/proto", true},
		{"adapter imports ui", "request/adapter/proto", "request/ui/calltrace", true},
		{"adapter imports port", "request/adapter/proto", "request/port", false},
		{"adapter imports controller", "request/adapter/proto", "request/controller/assert", false},
		{"other domain model", "collection/adapter/store", "request/model", false},
		{"other domain ui", "settings/ui/settingsui", "response/ui/respview", true},
		{"other domain adapter", "collection/adapter/openapi", "request/adapter/proto", true},
		{"other domain controller", "collection/ui/tree", "request/controller/assert", true},
		{"legacy package is not checked", "request/model", "textfmt", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := violations(imports{tc.from: {tc.to}}, nil)
			if (len(got) > 0) != tc.bad {
				t.Fatalf("%s -> %s: violations %q, want violation=%v", tc.from, tc.to, got, tc.bad)
			}
		})
	}
}

func TestAnExceptionSilencesItsViolationOnly(t *testing.T) {
	all := imports{"request/ui/calltrace": {"request/adapter/proto", "request/adapter/proto/ws"}}
	got := violations(all, map[string]bool{"request/ui/calltrace -> request/adapter/proto": true})
	if len(got) != 1 || !strings.Contains(got[0], "proto/ws") {
		t.Fatalf("violations = %q", got)
	}
}

func TestTheTreeFollowsTheLayerRules(t *testing.T) {
	if got := violations(scan(t), exceptions); len(got) > 0 {
		t.Fatalf("layer violations:\n%s", strings.Join(got, "\n"))
	}
}

// scan reads the imports of every non-test file under internal/.
func scan(t *testing.T) imports {
	t.Helper()
	all := imports{}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		for _, spec := range file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if rest, ok := strings.CutPrefix(path, modulePrefix); ok && !slices.Contains(all[dir], rest) {
				all[dir] = append(all[dir], rest)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return all
}
