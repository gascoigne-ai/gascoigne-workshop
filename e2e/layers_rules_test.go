//go:build !release

package e2e

import "strings"

// zone is the layer a package belongs to, read from its path alone (see
// internal/CLAUDE.md, the one place the layout is written).
type zone int

const (
	zoneUnclassified zone = iota // under internal/ but in no known zone: a violation
	zoneUnchecked                // cmd, e2e and anything outside internal and uitoolkit
	zoneKernel
	zoneFeatureRoot
	zoneController
	zoneFeatureUI
	zoneAdapter
	zoneBase
	zoneUILogic
	zoneUI
	zoneApp
	zoneExampleServer
	zoneToolkit
)

func (z zone) String() string {
	names := map[zone]string{
		zoneUnclassified: "unclassified", zoneUnchecked: "unchecked", zoneKernel: "kernel",
		zoneFeatureRoot: "feature root", zoneController: "controller", zoneFeatureUI: "feature ui",
		zoneBase: "base", zoneUILogic: "ui logic", zoneUI: "ui",
		zoneAdapter: "adapter", zoneApp: "app",
		zoneExampleServer: "exampleserver", zoneToolkit: "toolkit",
	}
	return names[z]
}

const (
	featurePrefix = "internal/feature/"
	baseFeature   = "request" // the one base feature every other may import
	uiPrefix      = "internal/ui/"
	uiLogicDir    = "logic"
)

// zoneOf classifies a module-relative package path. The feature is the
// directory name under internal/feature, empty for zones without one.
func zoneOf(relPath string) (zone, string) {
	switch {
	case relPath == "uitoolkit" || strings.HasPrefix(relPath, "uitoolkit/"):
		return zoneToolkit, ""
	case relPath == "internal/app" || strings.HasPrefix(relPath, "internal/app/"):
		return zoneApp, ""
	case relPath == "internal/exampleserver" || strings.HasPrefix(relPath, "internal/exampleserver/"):
		return zoneExampleServer, ""
	case strings.HasPrefix(relPath, "internal/kernel/"):
		return zoneKernel, ""
	case relPath == "internal/storage" || strings.HasPrefix(relPath, "internal/storage/"):
		return zoneBase, ""
	case relPath == "internal/ui" || strings.HasPrefix(relPath, uiPrefix):
		return uiZone(relPath), ""
	case strings.HasPrefix(relPath, featurePrefix):
		return featureZone(strings.TrimPrefix(relPath, featurePrefix))
	case strings.HasPrefix(relPath, "internal/"):
		return zoneUnclassified, ""
	}
	return zoneUnchecked, ""
}

// uiZone separates the toolkit-free logic packages from the views: a path
// segment named logic makes it uilogic, so it is matched before ui.
func uiZone(relPath string) zone {
	for _, segment := range strings.Split(relPath, "/") {
		if segment == uiLogicDir {
			return zoneUILogic
		}
	}
	return zoneUI
}

func featureZone(inFeature string) (zone, string) {
	feature, rest, nested := strings.Cut(inFeature, "/")
	switch {
	case feature == baseFeature:
		return zoneBase, ""
	case !nested:
		return zoneFeatureRoot, feature
	case rest == "ui" || strings.HasPrefix(rest, "ui/"):
		return zoneFeatureUI, feature
	case rest == "adapter" || strings.HasPrefix(rest, "adapter/"):
		return zoneAdapter, feature
	}
	return zoneController, feature
}

// selector picks packages by zone or by exact path.
type selector struct {
	Zones []zone
	Paths []string
}

// layerRule allows one kind of import. Own limits it to a target in the
// importer's own feature. Rules are default-deny: an import no row allows is
// a violation. The first matching row wins.
type layerRule struct {
	Name string
	From selector
	To   selector
	Own  bool
	// TestOnly limits the row to imports from test files; a production
	// import matching only such a row is a violation.
	TestOnly bool
	Reason   string
	// Below limits the row to a target below the importer's own path: a
	// parent importing its children.
	Below bool
}

// currentLayoutRules is the data form of internal/CLAUDE.md, Imports, for the
// layout the tree has today. The rows after the zone rules are the written
// exceptions, each with its reason.
var currentLayoutRules = []layerRule{
	{"feature root -> kernel", selector{Zones: []zone{zoneFeatureRoot}}, selector{Zones: []zone{zoneKernel}}, false, false, "a feature builds on the shared model", false},
	{"feature root -> own controllers", selector{Zones: []zone{zoneFeatureRoot}}, selector{Zones: []zone{zoneController}}, true, false, "a root may delegate to a sub-package of its own feature", false},
	{"controller -> kernel", selector{Zones: []zone{zoneController}}, selector{Zones: []zone{zoneKernel}}, false, false, "a feature builds on the shared model", false},
	{"controller -> own root, controllers", selector{Zones: []zone{zoneController}}, selector{Zones: []zone{zoneFeatureRoot, zoneController}}, true, false, "controllers share their feature's types", false},
	{"ui -> kernel, toolkit", selector{Zones: []zone{zoneFeatureUI}}, selector{Zones: []zone{zoneKernel, zoneToolkit}}, false, false, "views draw with the toolkit", false},
	{"ui -> own root, controllers", selector{Zones: []zone{zoneFeatureUI}}, selector{Zones: []zone{zoneFeatureRoot, zoneController}}, true, false, "views sit behind their feature's types", false},
	{"adapter -> kernel", selector{Zones: []zone{zoneAdapter}}, selector{Zones: []zone{zoneKernel}}, false, false, "a feature builds on the shared model", false},
	{"adapter -> own root, controllers, adapters", selector{Zones: []zone{zoneAdapter}}, selector{Zones: []zone{zoneFeatureRoot, zoneController, zoneAdapter}}, true, false, "adapters implement their feature's ports", false},
	{"app -> everything", selector{Zones: []zone{zoneApp}}, selector{Zones: allZones()}, false, false, "the composition root knows both sides", false},
	{"toolkit -> toolkit", selector{Zones: []zone{zoneToolkit}}, selector{Zones: []zone{zoneToolkit}}, false, false, "the toolkit knows nothing of the app", false},
	{"exampleserver -> kernel", selector{Zones: []zone{zoneExampleServer}}, selector{Zones: []zone{zoneKernel}}, false, false, "a dev tool uses the shared model", false},
	{"exampleserver -> ws", selector{Paths: []string{"internal/exampleserver"}}, selector{Paths: []string{"internal/feature/send/proto/ws"}}, false, false, "written exception: the dev server speaks the WebSocket framing the client uses", false},
	{"settings/theme -> toolkit core", selector{Paths: []string{"internal/feature/settings/theme"}}, selector{Paths: []string{"uitoolkit/core", "uitoolkit/theming"}}, false, false, "written exception: theme data is toolkit-bound", false},
	{"test only: exampleserver -> collections", selector{Paths: []string{"internal/exampleserver"}}, selector{Paths: []string{"internal/feature/collections"}}, false, true, "test fixture: the dev server tests load the real sample collection", false},
	{"test only: exampleserver -> proto", selector{Paths: []string{"internal/exampleserver"}}, selector{Paths: []string{"internal/feature/send/proto"}}, false, true, "test fixture: the dev server tests drive the real protocol registry", false},
	{"test only: history/ui/sentlog -> adapter", selector{Paths: []string{"internal/feature/history/ui/sentlog"}}, selector{Paths: []string{"internal/feature/history/adapter"}}, false, true, "test fixture: the view test uses the real history adapter", false},
	{"test only: settings/ui/formpage -> settingsfile", selector{Paths: []string{"internal/feature/settings/ui/formpage"}}, selector{Paths: []string{"internal/feature/settings/adapter/settingsfile"}}, false, true, "test fixture: the view test uses the real settings file adapter", false},
	{"test only: settings/ui/settingsui -> settingsfile", selector{Paths: []string{"internal/feature/settings/ui/settingsui"}}, selector{Paths: []string{"internal/feature/settings/adapter/settingsfile"}}, false, true, "test fixture: the dialog tests use the real settings file adapter", false},
}

// targetLayoutRules are the rows of the layout in .harness/reorg.md. They sit
// beside the current rows during the migration, so both layouts pass; a row
// may match nothing until its packages move (TestEveryRuleMatchesSomething
// exempts them). The reorganization's last task drops the old rows.
var targetLayoutRules = []layerRule{
	{"feature root -> base", selector{Zones: []zone{zoneFeatureRoot}}, selector{Zones: []zone{zoneBase}}, false, false, "a feature builds on the base feature and storage", false},
	{"controller -> base", selector{Zones: []zone{zoneController}}, selector{Zones: []zone{zoneBase}}, false, false, "a feature builds on the base feature and storage", false},
	{"adapter -> base", selector{Zones: []zone{zoneAdapter}}, selector{Zones: []zone{zoneBase}}, false, false, "a feature builds on the base feature and storage", false},
	{"uilogic -> base, root, controllers", selector{Zones: []zone{zoneUILogic}}, selector{Zones: []zone{zoneBase, zoneFeatureRoot, zoneController}}, false, false, "view logic decides over feature types and never draws", false},
	{"ui -> base, root, controllers, toolkit", selector{Zones: []zone{zoneUI}}, selector{Zones: []zone{zoneBase, zoneFeatureRoot, zoneController, zoneToolkit}}, false, false, "views read feature types and draw with the toolkit", false},
	{"ui -> format, theme, sizes", selector{Zones: []zone{zoneUI}}, selector{Paths: []string{"internal/ui/format", "internal/ui/theme", "internal/ui/sizes"}}, false, false, "the shared look and text helpers every view uses", false},
	{"ui -> own children", selector{Zones: []zone{zoneUI}}, selector{Zones: []zone{zoneUI, zoneUILogic}}, false, false, "a parent wires its children and their logic; siblings and parents stay out", true},
	{"exampleserver -> base", selector{Zones: []zone{zoneExampleServer}}, selector{Zones: []zone{zoneBase}}, false, false, "a dev tool uses the base model", false},
	{"exampleserver -> protocol ws", selector{Paths: []string{"internal/exampleserver"}}, selector{Paths: []string{"internal/feature/send/adapter/protocol/ws"}}, false, false, "written exception: the dev server speaks the WebSocket framing the client uses", false},
}

// layerRules is what the layer check applies: the current rows, then the
// target rows.
var layerRules = append(append([]layerRule{}, currentLayoutRules...), targetLayoutRules...)

func allZones() []zone {
	return []zone{zoneUnclassified, zoneUnchecked, zoneKernel, zoneFeatureRoot, zoneController, zoneFeatureUI, zoneAdapter, zoneBase, zoneUILogic, zoneUI, zoneApp, zoneExampleServer, zoneToolkit}
}
