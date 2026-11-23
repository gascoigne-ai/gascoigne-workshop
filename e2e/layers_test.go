//go:build !release

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// listedPackage is the part of `go list -json` the layer check reads.
type listedPackage struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
	Error        *struct{ Err string }
}

// layerEdge is one import between two module packages, with the files it
// comes from (code or test) for the message.
type layerEdge struct {
	From, To string
	Kind     string
}

// moduleEdges lists the direct imports of every package that stay inside the
// module, as module-relative paths. An import of the package itself (an
// external test package) is not an edge.
func moduleEdges(module string, pkg listedPackage) []layerEdge {
	from := relativeTo(module, pkg.ImportPath)
	var edges []layerEdge
	add := func(kind string, imports []string) {
		for _, imported := range imports {
			if imported != module && !strings.HasPrefix(imported, module+"/") {
				continue
			}
			to := relativeTo(module, imported)
			if to != from {
				edges = append(edges, layerEdge{From: from, To: to, Kind: kind})
			}
		}
	}
	add("import", pkg.Imports)
	add("test import", pkg.TestImports)
	add("external test import", pkg.XTestImports)
	return edges
}

func relativeTo(module, importPath string) string {
	return strings.TrimPrefix(strings.TrimPrefix(importPath, module), "/")
}

// matches tells whether the selector covers the package.
func (sel selector) matches(relPath string) bool {
	got, _ := zoneOf(relPath)
	for _, want := range sel.Zones {
		if got == want {
			return true
		}
	}
	for _, path := range sel.Paths {
		if path == relPath {
			return true
		}
	}
	return false
}

func (rule layerRule) allows(edge layerEdge) bool {
	if rule.TestOnly && !edge.fromTest() {
		return false
	}
	if !rule.From.matches(edge.From) || !rule.To.matches(edge.To) {
		return false
	}
	if rule.Below && !strings.HasPrefix(edge.To, edge.From+"/") {
		return false
	}
	if !rule.Own {
		return true
	}
	_, fromFeature := zoneOf(edge.From)
	_, toFeature := zoneOf(edge.To)
	return fromFeature != "" && fromFeature == toFeature
}

func (edge layerEdge) fromTest() bool { return edge.Kind != "import" }

// ruleFor returns the first row that allows the edge, or -1.
func ruleFor(rules []layerRule, edge layerEdge) int {
	for i, rule := range rules {
		if rule.allows(edge) {
			return i
		}
	}
	return -1
}

// checkLayers returns one message per forbidden import and per package in no
// zone. Packages outside zones that are checked (cmd, e2e) are skipped.
func checkLayers(module string, rules []layerRule, packages []listedPackage) []string {
	var violations []string
	for _, pkg := range packages {
		from := relativeTo(module, pkg.ImportPath)
		fromZone, _ := zoneOf(from)
		if fromZone == zoneUnclassified {
			violations = append(violations, from+": package is in no zone (internal/CLAUDE.md, Layout)")
		}
		if fromZone == zoneUnchecked || fromZone == zoneUnclassified {
			continue
		}
		for _, edge := range moduleEdges(module, pkg) {
			if ruleFor(rules, edge) < 0 {
				toZone, _ := zoneOf(edge.To)
				violations = append(violations, fmt.Sprintf("%s (%s) -> %s (%s): %s not allowed", edge.From, fromZone, edge.To, toZone, edge.Kind))
			}
		}
	}
	return violations
}

// goList runs go list in the repo root and decodes the concatenated JSON
// objects it prints. A failed run or a package with an error fails the test
// with go's own stderr.
func goList(t *testing.T, args ...string) []byte {
	t.Helper()
	command := exec.Command("go", args...)
	command.Dir = ".."
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes()
}

func listPackages(t *testing.T, tags string) []listedPackage {
	t.Helper()
	args := []string{"list"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, "-json=ImportPath,Imports,TestImports,XTestImports,Error", "./internal/...", "./uitoolkit/...")
	decoder := json.NewDecoder(bytes.NewReader(goList(t, args...)))
	var packages []listedPackage
	for {
		var pkg listedPackage
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			return packages
		}
		if err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		if pkg.Error != nil {
			t.Fatalf("go list: package %s: %s", pkg.ImportPath, pkg.Error.Err)
		}
		packages = append(packages, pkg)
	}
}

func moduleName(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(string(goList(t, "list", "-m")))
}

// layerTagSets are the build tag sets whose import graph is checked: the
// release build swaps files, so its imports can differ.
var layerTagSets = []string{"", "release"}

const testModule = "example.com/m"

func synthetic(importPath string, imports ...string) listedPackage {
	full := make([]string, len(imports))
	for i, imported := range imports {
		full[i] = testModule + "/" + imported
	}
	return listedPackage{ImportPath: testModule + "/" + importPath, Imports: full}
}

func TestCheckLayersRejects(t *testing.T) {
	xtest := synthetic("internal/feature/send/requestedit")
	xtest.XTestImports = []string{testModule + "/internal/feature/history"}
	cases := []struct {
		name string
		pkg  listedPackage
	}{
		{"format imports an adapter", synthetic("internal/ui/format", "internal/feature/send/adapter/preview")},
		{"feature root imports another feature root", synthetic("internal/feature/send", "internal/feature/history")},
		{"controller external test imports another feature", xtest},
		{"ui imports an adapter", synthetic("internal/feature/send/ui/respview", "internal/feature/send/adapter/preview")},
		{"ui imports another ui", synthetic("internal/feature/send/ui/respview", "internal/feature/send/ui/calltrace")},
		{"controller imports a toolkit widget", synthetic("internal/feature/send/requestedit", "uitoolkit/widget")},
		{"adapter imports a ui", synthetic("internal/feature/history/adapter/historyfile", "internal/feature/history/ui/sentlog")},
		{"toolkit imports internal", synthetic("uitoolkit/widget", "internal/feature/request")},
		{"exampleserver imports the send root", synthetic("internal/exampleserver", "internal/feature/send")},
		{"settings theme imports a toolkit widget", synthetic("internal/feature/settings/theme", "uitoolkit/widget")},
		{"unclassified internal package", synthetic("internal/stuff")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkLayers(testModule, layerRules, []listedPackage{tc.pkg}); len(got) == 0 {
				t.Fatal("checkLayers found no violation")
			}
		})
	}
}

func TestCheckLayersAllows(t *testing.T) {
	ownXtest := synthetic("internal/feature/send/requestedit")
	ownXtest.XTestImports = []string{testModule + "/internal/feature/send/requestedit", testModule + "/internal/feature/send"}
	cases := []struct {
		name string
		pkg  listedPackage
	}{
		{"theme imports toolkit core and theming", synthetic("internal/feature/settings/theme", "uitoolkit/core", "uitoolkit/theming")},
		{"app imports anything", synthetic("internal/app", "internal/feature/send", "uitoolkit/widget", "internal/feature/send/ui/respview")},
		{"cmd is not checked", synthetic("cmd/workshop", "internal/app", "internal/feature/send")},
		{"e2e is not checked", synthetic("e2e", "internal/feature/send")},
		{"standard library and ebiten are ignored", listedPackage{ImportPath: testModule + "/internal/feature/request/severity", Imports: []string{"fmt", "github.com/hajimehoshi/ebiten/v2"}}},
		{"external test imports its own package and root", ownXtest},
		{"workspace root imports its controller", synthetic("internal/feature/workspace", "internal/feature/workspace/state")},
		{"adapter imports its own root", synthetic("internal/feature/workspace/adapter/folders", "internal/feature/workspace")},
		{"ui imports toolkit and own controller", synthetic("internal/feature/send/ui/respview", "uitoolkit/widget", "internal/feature/send/viewer", "internal/feature/send")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkLayers(testModule, layerRules, []listedPackage{tc.pkg}); len(got) != 0 {
				t.Fatalf("checkLayers = %v, want none", got)
			}
		})
	}
}

func TestCheckLayersTargetLayout(t *testing.T) {
	cases := []struct {
		name    string
		pkg     listedPackage
		allowed bool
	}{
		{"feature root imports base", synthetic("internal/feature/send", "internal/feature/request", "internal/storage"), true},
		{"controller imports base", synthetic("internal/feature/send/assert", "internal/feature/request"), true},
		{"adapter imports base", synthetic("internal/feature/history/adapter/historyfile", "internal/storage", "internal/feature/request"), true},
		{"base imports a feature", synthetic("internal/feature/request", "internal/feature/send"), false},
		{"uilogic imports base, root, controller", synthetic("internal/ui/sidebar/logic", "internal/feature/request", "internal/feature/collections", "internal/feature/send/assert"), true},
		{"uilogic imports toolkit", synthetic("internal/ui/sidebar/logic", "uitoolkit/core"), false},
		{"uilogic imports ui", synthetic("internal/ui/sidebar/logic", "internal/ui/theme"), false},
		{"ui imports base, root, controller, toolkit", synthetic("internal/ui/sidebar", "internal/feature/request", "internal/feature/collections", "internal/feature/send/assert", "uitoolkit/widget"), true},
		{"ui imports format, theme, sizes", synthetic("internal/ui/sidebar", "internal/ui/format", "internal/ui/theme", "internal/ui/sizes"), true},
		{"ui imports its own logic", synthetic("internal/ui/sidebar", "internal/ui/sidebar/logic"), true},
		{"ui parent imports child", synthetic("internal/ui/editor", "internal/ui/editor/urlbar", "internal/ui/editor/mcp/mcpform"), true},
		{"ui imports sibling", synthetic("internal/ui/sidebar", "internal/ui/header"), false},
		{"ui child imports parent", synthetic("internal/ui/editor/urlbar", "internal/ui/editor"), false},
		{"ui imports an adapter", synthetic("internal/ui/sidebar", "internal/feature/history/adapter/historyfile"), false},
		{"ui imports a prefix lookalike", synthetic("internal/ui/side", "internal/ui/sidebar"), false},
		{"adapter imports ui", synthetic("internal/feature/history/adapter/historyfile", "internal/ui/theme"), false},
		{"feature root imports ui", synthetic("internal/feature/send", "internal/ui/theme"), false},
		{"app imports base, ui, uilogic", synthetic("internal/app", "internal/feature/request", "internal/storage", "internal/ui/window", "internal/ui/sidebar/logic"), true},
		{"exampleserver imports base", synthetic("internal/exampleserver", "internal/feature/request"), true},
		{"exampleserver imports protocol ws", synthetic("internal/exampleserver", "internal/feature/send/adapter/protocol/ws"), true},
		{"exampleserver imports another adapter", synthetic("internal/exampleserver", "internal/feature/send/adapter/preview"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkLayers(testModule, layerRules, []listedPackage{tc.pkg})
			if tc.allowed && len(got) != 0 {
				t.Fatalf("checkLayers = %v, want none", got)
			}
			if !tc.allowed && len(got) == 0 {
				t.Fatal("checkLayers found no violation")
			}
		})
	}
}

func TestCheckLayersTestOnlyRows(t *testing.T) {
	production := synthetic("internal/feature/history/ui/sentlog", "internal/feature/history/adapter/historyfile")
	test := synthetic("internal/feature/history/ui/sentlog")
	test.TestImports = []string{testModule + "/internal/feature/history/adapter/historyfile"}
	if got := checkLayers(testModule, layerRules, []listedPackage{production}); len(got) == 0 {
		t.Error("a production import matching only a test-only row must be a violation")
	}
	if got := checkLayers(testModule, layerRules, []listedPackage{test}); len(got) != 0 {
		t.Errorf("a test import matching a test-only row must pass, got %v", got)
	}
}

func TestZoneOf(t *testing.T) {
	cases := []struct {
		path        string
		wantZone    zone
		wantFeature string
	}{
		{"internal/ui/format", zoneUI, ""},
		{"internal/feature/request/severity", zoneBase, ""},
		{"internal/feature/send", zoneFeatureRoot, "send"},
		{"internal/feature/send/ui/respview", zoneFeatureUI, "send"},
		{"internal/feature/send/adapter/preview", zoneAdapter, "send"},
		{"internal/feature/request", zoneBase, ""},
		{"internal/feature/request/sub", zoneBase, ""},
		{"internal/storage", zoneBase, ""},
		{"internal/ui/theme", zoneUI, ""},
		{"internal/ui/editor/urlbar", zoneUI, ""},
		{"internal/ui/sidebar/logic", zoneUILogic, ""},
		{"internal/ui/editor/kvtable/logic", zoneUILogic, ""},
		{"internal/ui/logicbox", zoneUI, ""},
		{"internal/feature/send/adapter/protocol/ws", zoneAdapter, "send"},
		{"internal/platform/fonts", zoneUnclassified, ""}, // the platform layer is gone: a new package goes into a feature or the toolkit
		{"internal/app", zoneApp, ""},
		{"internal/exampleserver", zoneExampleServer, ""},
		{"uitoolkit/core", zoneToolkit, ""},
		{"cmd/workshop", zoneUnchecked, ""},
		{"e2e", zoneUnchecked, ""},
		{"internal/stuff", zoneUnclassified, ""},
	}
	for _, tc := range cases {
		gotZone, gotFeature := zoneOf(tc.path)
		if gotZone != tc.wantZone || gotFeature != tc.wantFeature {
			t.Errorf("zoneOf(%q) = %v, %q; want %v, %q", tc.path, gotZone, gotFeature, tc.wantZone, tc.wantFeature)
		}
	}
}

func TestRuleTableHasReasons(t *testing.T) {
	for _, rule := range layerRules {
		if rule.Name == "" || rule.Reason == "" {
			t.Errorf("rule %q needs a name and a reason", rule.Name)
		}
	}
}

func TestEveryRuleMatchesSomething(t *testing.T) {
	module := moduleName(t)
	used := make([]bool, len(layerRules))
	for _, tags := range layerTagSets {
		for _, pkg := range listPackages(t, tags) {
			for _, edge := range moduleEdges(module, pkg) {
				if i := ruleFor(layerRules, edge); i >= 0 {
					used[i] = true
				}
			}
		}
	}
	for i, rule := range layerRules {
		pendingMigration := i >= len(currentLayoutRules)
		if !used[i] && !pendingMigration {
			t.Errorf("rule %q matches no import in the tree: drop it or fix its selector", rule.Name)
		}
	}
}

func TestNoUnclassifiedPackage(t *testing.T) {
	module := moduleName(t)
	for _, pkg := range listPackages(t, "") {
		if got, _ := zoneOf(relativeTo(module, pkg.ImportPath)); got == zoneUnclassified {
			t.Errorf("%s is in no zone: put it in a feature, storage or ui (internal/CLAUDE.md)", pkg.ImportPath)
		}
	}
}

// The request model and OpenCollection live in the one base feature; the
// kernel is gone.
func TestRequestModelIsTheBaseFeature(t *testing.T) {
	module := moduleName(t)
	listed := map[string]bool{}
	for _, pkg := range listPackages(t, "") {
		listed[relativeTo(module, pkg.ImportPath)] = true
	}
	if !listed["internal/feature/request"] {
		t.Error("internal/feature/request is not in the tree")
	}
	for _, gone := range []string{"internal/kernel/request", "internal/kernel/openset"} {
		if listed[gone] {
			t.Errorf("%s still exists: it moved to internal/feature/request", gone)
		}
	}
}

func TestSeverityIsABaseFeatureLeaf(t *testing.T) {
	module := moduleName(t)
	listed := map[string]bool{}
	for _, pkg := range listPackages(t, "") {
		listed[relativeTo(module, pkg.ImportPath)] = true
	}
	if !listed["internal/feature/request/severity"] {
		t.Error("internal/feature/request/severity is not in the tree")
	}
	if listed["internal/kernel/severity"] {
		t.Error("internal/kernel/severity still exists: it moved under the base feature")
	}
}

func TestStorageIsABasePackage(t *testing.T) {
	module := moduleName(t)
	listed := map[string]bool{}
	for _, pkg := range listPackages(t, "") {
		listed[relativeTo(module, pkg.ImportPath)] = true
	}
	if !listed["internal/storage"] {
		t.Error("internal/storage is not in the tree")
	}
	for _, gone := range []string{"internal/kernel/userdir", "internal/kernel/fileversion"} {
		if listed[gone] {
			t.Errorf("%s still exists: it merged into internal/storage", gone)
		}
	}
}

// textfmt and jsonview merged into ui/format; with them gone the kernel
// holds nothing, so no package may live under it.
func TestFormatIsAUIPackageAndKernelIsGone(t *testing.T) {
	module := moduleName(t)
	listed := map[string]bool{}
	for _, pkg := range listPackages(t, "") {
		relPath := relativeTo(module, pkg.ImportPath)
		listed[relPath] = true
		if strings.HasPrefix(relPath, "internal/kernel") {
			t.Errorf("%s still exists: internal/kernel is gone", relPath)
		}
	}
	if !listed["internal/ui/format"] {
		t.Error("internal/ui/format is not in the tree")
	}
}

// The protocol clients do network and process I/O and implement
// request.Sender: an adapter of the send feature, not a controller.
func TestProtocolClientsAreASendAdapter(t *testing.T) {
	module := moduleName(t)
	listed := map[string]bool{}
	for _, pkg := range listPackages(t, "") {
		listed[relativeTo(module, pkg.ImportPath)] = true
	}
	for _, want := range []string{"internal/feature/send/adapter/protocol", "internal/feature/send/adapter/protocol/ws"} {
		if !listed[want] {
			t.Errorf("%s is not in the tree", want)
		}
	}
	if listed["internal/feature/send/proto"] {
		t.Error("internal/feature/send/proto still exists: it moved to send/adapter/protocol")
	}
}

// The adapters are named for what they hold: history/adapter was package
// "history" (a clash with the feature root) and collections/adapter/store said
// nothing at an import site.
func TestHistoryAndCollectionAdaptersAreNamedForWhatTheyHold(t *testing.T) {
	module := moduleName(t)
	listed := map[string]bool{}
	for _, pkg := range listPackages(t, "") {
		listed[relativeTo(module, pkg.ImportPath)] = true
	}
	for _, want := range []string{"internal/feature/history/adapter/historyfile", "internal/feature/collections/adapter/collectionstore"} {
		if !listed[want] {
			t.Errorf("%s is not in the tree", want)
		}
	}
	for _, gone := range []string{"internal/feature/collections/adapter/store", "internal/feature/history/adapter"} {
		if listed[gone] {
			t.Errorf("%s still holds a package: it moved into a named sub-directory", gone)
		}
	}
}

func TestRequestEditIsNamedForWhatItHolds(t *testing.T) {
	module := moduleName(t)
	listed := map[string]bool{}
	for _, pkg := range listPackages(t, "") {
		listed[relativeTo(module, pkg.ImportPath)] = true
	}
	if !listed["internal/feature/send/requestedit"] {
		t.Error("internal/feature/send/requestedit is not in the tree")
	}
	if listed["internal/feature/send/editor"] {
		t.Error("internal/feature/send/editor still holds a package: it is renamed requestedit")
	}
}

func TestRealTreeObeysLayers(t *testing.T) {
	module := moduleName(t)
	for _, tags := range layerTagSets {
		for _, violation := range checkLayers(module, layerRules, listPackages(t, tags)) {
			t.Errorf("tags %q: %s", tags, violation)
		}
	}
}
