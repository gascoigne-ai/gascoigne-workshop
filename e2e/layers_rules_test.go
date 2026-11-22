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
	zoneUI
	zoneAdapter
	zonePlatform
	zoneApp
	zoneExampleServer
	zoneToolkit
)

func (z zone) String() string {
	names := map[zone]string{
		zoneUnclassified: "unclassified", zoneUnchecked: "unchecked", zoneKernel: "kernel",
		zoneFeatureRoot: "feature root", zoneController: "controller", zoneUI: "ui",
		zoneAdapter: "adapter", zonePlatform: "platform", zoneApp: "app",
		zoneExampleServer: "exampleserver", zoneToolkit: "toolkit",
	}
	return names[z]
}

const (
	featurePrefix = "internal/feature/"
	platformDir   = "internal/platform/"
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
	case strings.HasPrefix(relPath, platformDir):
		return zonePlatform, ""
	case strings.HasPrefix(relPath, featurePrefix):
		return featureZone(strings.TrimPrefix(relPath, featurePrefix))
	case strings.HasPrefix(relPath, "internal/"):
		return zoneUnclassified, ""
	}
	return zoneUnchecked, ""
}

func featureZone(inFeature string) (zone, string) {
	feature, rest, nested := strings.Cut(inFeature, "/")
	switch {
	case !nested:
		return zoneFeatureRoot, feature
	case rest == "ui" || strings.HasPrefix(rest, "ui/"):
		return zoneUI, feature
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
}

var (
	settingsPlatform  = []string{"internal/platform/displayscale", "internal/platform/fonts", "internal/platform/keyrepeat", "internal/platform/systeminfo"}
	workspacePlatform = []string{"internal/platform/filedialog", "internal/platform/folders", "internal/platform/systemclock"}
)

// layerRules is the data form of internal/CLAUDE.md, Imports. The rows after
// the zone rules are the written exceptions, each with its reason.
var layerRules = []layerRule{
	{"feature root -> kernel", selector{Zones: []zone{zoneFeatureRoot}}, selector{Zones: []zone{zoneKernel}}, false, false, "a feature builds on the shared model"},
	{"feature root -> own controllers", selector{Zones: []zone{zoneFeatureRoot}}, selector{Zones: []zone{zoneController}}, true, false, "a root may delegate to a sub-package of its own feature"},
	{"controller -> kernel", selector{Zones: []zone{zoneController}}, selector{Zones: []zone{zoneKernel}}, false, false, "a feature builds on the shared model"},
	{"controller -> own root, controllers", selector{Zones: []zone{zoneController}}, selector{Zones: []zone{zoneFeatureRoot, zoneController}}, true, false, "controllers share their feature's types"},
	{"ui -> kernel, toolkit", selector{Zones: []zone{zoneUI}}, selector{Zones: []zone{zoneKernel, zoneToolkit}}, false, false, "views draw with the toolkit"},
	{"ui -> own root, controllers", selector{Zones: []zone{zoneUI}}, selector{Zones: []zone{zoneFeatureRoot, zoneController}}, true, false, "views sit behind their feature's types"},
	{"adapter -> kernel", selector{Zones: []zone{zoneAdapter}}, selector{Zones: []zone{zoneKernel}}, false, false, "a feature builds on the shared model"},
	{"adapter -> own root, controllers, adapters", selector{Zones: []zone{zoneAdapter}}, selector{Zones: []zone{zoneFeatureRoot, zoneController, zoneAdapter}}, true, false, "adapters implement their feature's ports"},
	{"platform -> settings root", selector{Paths: settingsPlatform}, selector{Paths: []string{"internal/feature/settings"}}, false, false, "the platform adapter implements a port settings owns, so it imports the port types"},
	{"platform -> workspace root", selector{Paths: workspacePlatform}, selector{Paths: []string{"internal/feature/workspace"}}, false, false, "the platform adapter implements a port workspace owns, so it imports the port types"},
	{"app -> everything", selector{Zones: []zone{zoneApp}}, selector{Zones: allZones()}, false, false, "the composition root knows both sides"},
	{"toolkit -> toolkit", selector{Zones: []zone{zoneToolkit}}, selector{Zones: []zone{zoneToolkit}}, false, false, "the toolkit knows nothing of the app"},
	{"exampleserver -> kernel", selector{Zones: []zone{zoneExampleServer}}, selector{Zones: []zone{zoneKernel}}, false, false, "a dev tool uses the shared model"},
	{"exampleserver -> ws", selector{Paths: []string{"internal/exampleserver"}}, selector{Paths: []string{"internal/feature/send/proto/ws"}}, false, false, "written exception: the dev server speaks the WebSocket framing the client uses"},
	{"settings/theme -> toolkit core", selector{Paths: []string{"internal/feature/settings/theme"}}, selector{Paths: []string{"uitoolkit/core", "uitoolkit/theming"}}, false, false, "written exception: theme data is toolkit-bound"},
	{"test only: exampleserver -> collections", selector{Paths: []string{"internal/exampleserver"}}, selector{Paths: []string{"internal/feature/collections"}}, false, true, "test fixture: the dev server tests load the real sample collection"},
	{"test only: exampleserver -> proto", selector{Paths: []string{"internal/exampleserver"}}, selector{Paths: []string{"internal/feature/send/proto"}}, false, true, "test fixture: the dev server tests drive the real protocol registry"},
	{"test only: history/ui/sentlog -> adapter", selector{Paths: []string{"internal/feature/history/ui/sentlog"}}, selector{Paths: []string{"internal/feature/history/adapter"}}, false, true, "test fixture: the view test uses the real history adapter"},
	{"test only: settings/ui/formpage -> fonts", selector{Paths: []string{"internal/feature/settings/ui/formpage"}}, selector{Paths: []string{"internal/platform/fonts"}}, false, true, "test fixture: the view test uses the real font store"},
	{"test only: settings/ui/formpage -> settingsfile", selector{Paths: []string{"internal/feature/settings/ui/formpage"}}, selector{Paths: []string{"internal/feature/settings/adapter/settingsfile"}}, false, true, "test fixture: the view test uses the real settings file adapter"},
	{"test only: settings/ui/settingsui -> settingsfile", selector{Paths: []string{"internal/feature/settings/ui/settingsui"}}, selector{Paths: []string{"internal/feature/settings/adapter/settingsfile"}}, false, true, "test fixture: the dialog tests use the real settings file adapter"},
}

func allZones() []zone {
	return []zone{zoneUnclassified, zoneUnchecked, zoneKernel, zoneFeatureRoot, zoneController, zoneUI, zoneAdapter, zonePlatform, zoneApp, zoneExampleServer, zoneToolkit}
}
