package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSettingsMenuOpensChangesThemeAndSaves(t *testing.T) {
	a := testApp(t)
	path := a.SettingsPath()
	click(a, menuBtnX, menuBtnY)
	if !a.menu {
		t.Fatal("the three-dot button must open the menu")
	}

	a.prefs.Search = widget.NewTextField("theme") // brings the Theme setting to the top
	clickText(t, a, "Light", 0)
	if a.settings.Theme != "light" || !a.menu {
		t.Fatalf("theme %q, menu open %v (choosing must keep it open)", a.settings.Theme, a.menu)
	}
	if a.ctx.Th.Color(core.Bg) != theming.Light().Color(core.Bg) {
		t.Fatal("the live theme did not change")
	}
	saved, err := settingsDisk.Load(path)
	if err != nil || saved.Theme != "light" {
		t.Fatalf("saved %+v err %v", saved, err)
	}

	a.prefs.Search = widget.NewTextField("text size")
	clickText(t, a, "+", 0)
	clickText(t, a, "+", 0)
	if a.ctx.Th.FontSize != 16 {
		t.Fatalf("font size = %v, want 16", a.ctx.Th.FontSize)
	}
	if saved, _ := settingsDisk.Load(path); saved.FontSize != 16 || saved.Theme != "light" {
		t.Fatalf("saved %+v", saved)
	}
}

func TestSettingsSaveFailureIsReported(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: filepath.Join(blocker, "settings.json")})) // a file where a directory is needed
	if err != nil {
		t.Fatal(err)
	}
	a.changeSettings(func(prefs *settings.Settings) { prefs.Theme = "light" })
	if !strings.HasPrefix(a.status, "Could not save settings:") {
		t.Fatalf("status = %q", a.status)
	}
	if a.settings.Theme != "light" {
		t.Fatal("the change must still apply in memory")
	}
}

func TestBrokenSettingsFileFallsBackAndSaysSo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.settings, settings.Default()) || !strings.Contains(a.status, "parse settings") {
		t.Fatalf("settings %+v status %q", a.settings, a.status)
	}
}

func TestSettingsStartFromTheSavedChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := settingsDisk.Save(path, settings.Settings{Theme: "light", FontSize: 13}); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path, OpenMenu: true}))
	if err != nil {
		t.Fatal(err)
	}
	if a.ctx.Th.Color(core.Bg) != theming.Light().Color(core.Bg) || a.ctx.Th.FontSize != 13 || !a.menu {
		t.Fatalf("theme/size/menu not restored: font %v menu %v", a.ctx.Th.FontSize, a.menu)
	}
}

func TestCtrlLFocusesAndSelectsTheURL(t *testing.T) {
	a := testApp(t)
	click(a, 1100, 600) // click on empty space so nothing is focused
	a.step(combo(ebiten.KeyL, true, false, false))
	url := &a.cur().url
	if !url.Focused || url.SelectedText() != url.Text {
		t.Fatalf("focused %v selection %q", url.Focused, url.SelectedText())
	}
}

func TestCtrlTOpensAndCtrlWClosesTabs(t *testing.T) {
	a := testApp(t)
	a.step(combo(ebiten.KeyT, true, false, false))
	if len(a.list.Tabs) != 2 || a.list.Active != 1 {
		t.Fatalf("Ctrl+T: %d tabs, active %d", len(a.list.Tabs), a.list.Active)
	}
	a.step(combo(ebiten.KeyW, true, false, false))
	if len(a.list.Tabs) != 1 || a.list.Active != 0 || a.cur().Req.Name != "Search Repos" {
		t.Fatalf("Ctrl+W: %d tabs, active %d (%s)", len(a.list.Tabs), a.list.Active, a.cur().Req.Name)
	}
	a.step(combo(ebiten.KeyW, true, false, false))
	if len(a.list.Tabs) != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("closing the last tab must leave a blank request, got %d tabs (%s)", len(a.list.Tabs), a.cur().Req.Name)
	}
}

func TestCtrlTabSwitchesTabsAndWraps(t *testing.T) {
	a := testApp(t)
	a.newRequest()
	a.newRequest() // three tabs, the last active
	a.step(combo(ebiten.KeyTab, true, false, false))
	if a.list.Active != 0 {
		t.Fatalf("Ctrl+Tab from the last tab must wrap to the first, active %d", a.list.Active)
	}
	a.step(combo(ebiten.KeyTab, true, true, false))
	if a.list.Active != 2 {
		t.Fatalf("Ctrl+Shift+Tab must go back and wrap, active %d", a.list.Active)
	}
}

func TestCtrlCommaOpensTheSettingsMenu(t *testing.T) {
	a := testApp(t)
	a.step(combo(ebiten.KeyComma, true, false, false))
	if !a.menu {
		t.Fatal("Ctrl+, must open the settings menu")
	}
}

func TestShortcutsNeedTheirModifiers(t *testing.T) {
	a := testApp(t)
	for _, k := range []ebiten.Key{ebiten.KeyT, ebiten.KeyW, ebiten.KeyL, ebiten.KeyEnter, ebiten.KeyComma, ebiten.KeyTab} {
		a.step(combo(k, false, false, false)) // plain letters must never trigger anything
	}
	if len(a.list.Tabs) != 1 || a.menu || a.cur().Sending() {
		t.Fatalf("plain keys changed state: %d tabs, menu %v", len(a.list.Tabs), a.menu)
	}
}

func TestShortcutTableHasUniqueBindingsAndLabels(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range shortcuts {
		id := s.def.String()
		if s.name == "" || s.run == nil || seen[id] {
			t.Errorf("bad or duplicate shortcut %q (%q)", s.name, id)
		}
		seen[id] = true
	}
	if !seen["Ctrl+Enter"] || !seen["Ctrl+Shift+Tab"] {
		t.Fatalf("labels = %v", seen)
	}
}

// Below the layout chooser: Contrast chips at y=374, then Colors rows from
// y=560 (Selection), 588, 616, 644 (Accent); field at x=1170, Reset at
// x=1234; the "Reset colors" button under the last row.
const (
	colorRow0Y   = 256
	colorFieldX  = 589
	colorResetX  = 652
	resetColorsX = 496
	resetColorsY = 368
)

func typeColor(app *App, tok core.Token, row int, text string) {
	in := app.colors.Input(app, tok)
	click(app, colorFieldX, colorRow0Y+row*core.RowH)
	in.SelectAll()
	app.typeText(text)
}

func TestContrastChooserStrengthensTheColorsAndIsSaved(t *testing.T) {
	a, path := openSettingsMenu(t)
	a.prefs.Search = widget.NewTextField("contrast")
	before := core.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text))
	clickText(t, a, "High", 0)
	if a.settings.Contrast != "high" {
		t.Fatalf("Contrast = %q", a.settings.Contrast)
	}
	after := core.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text))
	if after < appearance.HighRatio-0.01 || after <= before {
		t.Fatalf("selection contrast %.2f -> %.2f", before, after)
	}
	if saved, _ := settingsDisk.Load(path); saved.Contrast != "high" {
		t.Fatalf("saved %+v", saved)
	}
	clickText(t, a, "Normal", 0)
	if a.settings.Contrast != "normal" || core.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text)) != before {
		t.Fatal("back to normal restores the default colors")
	}
}

func TestTypingAColorAppliesItLiveAndSavesIt(t *testing.T) {
	a, path := openSettingsPage(t, "colors")
	typeColor(a, core.Selection, 0, "#223344")
	if got := a.ctx.Th.Color(core.Selection); got != core.RGB(0x223344) {
		t.Fatalf("live color = %v", got)
	}
	saved, _ := settingsDisk.Load(path)
	if saved.Colors["dark"]["selection"] != "#223344" {
		t.Fatalf("saved %+v", saved.Colors)
	}
	if a.colors.Note(core.Selection) != "" {
		t.Fatalf("note = %q", a.colors.Note(core.Selection))
	}
}

func TestAnUnreadableColorIsRefusedWithItsRatio(t *testing.T) {
	a, path := openSettingsPage(t, "colors")
	def := a.ctx.Th.Color(core.FindMatch)
	typeColor(a, core.FindMatch, 1, "#ffffff") // white under light text
	note := a.colors.Note(core.FindMatch)
	if !strings.Contains(note, "below 4.5:1") || !strings.Contains(note, "1.") {
		t.Fatalf("note = %q: it must say how far off the contrast is", note)
	}
	if a.ctx.Th.Color(core.FindMatch) != def {
		t.Fatal("a refused color must not apply")
	}
	if saved, _ := settingsDisk.Load(path); len(saved.Colors["dark"]) != 0 {
		t.Fatalf("a refused color must not be saved: %+v", saved.Colors)
	}
	typeColor(a, core.FindMatch, 1, "zzz")
	if !strings.Contains(a.colors.Note(core.FindMatch), "#rrggbb") {
		t.Fatalf("note = %q", a.colors.Note(core.FindMatch))
	}
	typeColor(a, core.FindMatch, 1, "#334455")
	if a.colors.Note(core.FindMatch) != "" || a.ctx.Th.Color(core.FindMatch) != core.RGB(0x334455) {
		t.Fatalf("a readable color clears the note and applies: %q %v", a.colors.Note(core.FindMatch), a.ctx.Th.Color(core.FindMatch))
	}
}

func TestResetButtonsRestoreTheDefaults(t *testing.T) {
	a, path := openSettingsPage(t, "colors")
	def := a.ctx.Th.Color(core.Selection)
	typeColor(a, core.Selection, 0, "#223344")
	typeColor(a, core.FindMatch, 1, "#334455")
	click(a, colorResetX, colorRow0Y)
	if a.ctx.Th.Color(core.Selection) != def || a.colors.Input(a, core.Selection).Text != core.FormatHex(def) {
		t.Fatalf("selection %v, field %q", a.ctx.Th.Color(core.Selection), a.colors.Input(a, core.Selection).Text)
	}
	if a.ctx.Th.Color(core.FindMatch) != core.RGB(0x334455) {
		t.Fatal("resetting one color must leave the others")
	}
	click(a, resetColorsX, resetColorsY)
	if a.ctx.Th.Color(core.FindMatch) == core.RGB(0x334455) {
		t.Fatal("Reset colors drops every override")
	}
	if saved, _ := settingsDisk.Load(path); len(saved.Colors["dark"]) != 0 {
		t.Fatalf("saved %+v", saved.Colors)
	}
}

func TestSwitchingThemeShowsThatThemesColorsAndKeepsEachThemesOverrides(t *testing.T) {
	a, _ := openSettingsPage(t, "colors")
	typeColor(a, core.Selection, 0, "#223344")
	a.prefs.Cat, a.prefs.Search = "appearance", widget.NewTextField("theme")
	a.step(core.Input{})
	clickText(t, a, "Light", 0)
	if a.ctx.Th.Color(core.Selection) != appearance.ThemeFor("light", settings.DefaultFontSize).Color(core.Selection) {
		t.Fatal("the dark override must not leak into the light theme")
	}
	if a.colors.Input(a, core.Selection).Text != core.FormatHex(appearance.ThemeFor("light", settings.DefaultFontSize).Color(core.Selection)) {
		t.Fatalf("the field must show the light theme's color, got %q", a.colors.Input(a, core.Selection).Text)
	}
	clickText(t, a, "Dark", 0) // back to Dark
	if a.ctx.Th.Color(core.Selection) != core.RGB(0x223344) {
		t.Fatal("the dark override is remembered")
	}
}

func TestColorsAreRestoredAtStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := settings.Default()
	s.Contrast = "high"
	s.Colors = map[string]map[string]string{"dark": {"accent": "#aaccee"}}
	if err := settingsDisk.Save(path, s); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	if a.ctx.Th.Color(core.Accent) != core.RGB(0xaaccee) {
		t.Fatalf("accent = %v", a.ctx.Th.Color(core.Accent))
	}
	if core.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text)) < appearance.HighRatio-0.01 {
		t.Fatal("high contrast is applied at start")
	}
}

func TestShortcutLabelsReadNaturally(t *testing.T) {
	labels := map[string]string{}
	for _, s := range shortcuts {
		labels[s.name] = s.def.String()
	}
	want := map[string]string{"Move row up": "Alt+Up", "Move row down": "Alt+Down", "Delete row": "Delete", "Settings": "Ctrl+,", "Next tab": "Ctrl+Tab"}
	for name, label := range want {
		if labels[name] != label {
			t.Errorf("%s: %q, want %q", name, labels[name], label)
		}
	}
}

func TestRecordingANewBindingRebindsTheAction(t *testing.T) {
	a := testApp(t)
	path := a.SettingsPath()
	a.newRequest() // two tabs
	a.shortcutKeys.Start("close-tab")
	if a.shortcutKeys.Recording() != "close-tab" {
		t.Fatal("recording must start")
	}
	a.step(combo(ebiten.KeyQ, true, false, false))
	if a.shortcutKeys.Recording() != "" || a.settings.Shortcuts["close-tab"] != "Ctrl+Q" {
		t.Fatalf("recording %q shortcuts %v", a.shortcutKeys.Recording(), a.settings.Shortcuts)
	}
	if saved, _ := settingsDisk.Load(path); saved.Shortcuts["close-tab"] != "Ctrl+Q" {
		t.Fatalf("saved %+v", saved.Shortcuts)
	}
	a.step(combo(ebiten.KeyW, true, false, false)) // the old key does nothing now
	if len(a.list.Tabs) != 2 {
		t.Fatal("Ctrl+W must no longer close a tab")
	}
	a.step(combo(ebiten.KeyQ, true, false, false))
	if len(a.list.Tabs) != 1 {
		t.Fatal("Ctrl+Q closes a tab")
	}
	if shortcutpage.KeyText(a, "close-tab") != "Ctrl+Q" {
		t.Fatalf("text = %q", shortcutpage.KeyText(a, "close-tab"))
	}
}

func TestEscapeCancelsRecordingAndBackspaceUnbinds(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(core.Input{Escape: true})
	if a.shortcutKeys.Recording() != "" || len(a.settings.Shortcuts) != 0 || a.menu {
		t.Fatalf("recording %q shortcuts %v", a.shortcutKeys.Recording(), a.settings.Shortcuts)
	}

	a.newRequest()
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyBackspace, false, false, false))
	if v, ok := a.settings.Shortcuts["close-tab"]; !ok || v != "" {
		t.Fatalf("shortcuts = %v: Backspace stores an empty (unbound) binding", a.settings.Shortcuts)
	}
	a.step(combo(ebiten.KeyW, true, false, false))
	if len(a.list.Tabs) != 2 {
		t.Fatal("an unbound action fires nothing")
	}
	if shortcutpage.KeyText(a, "close-tab") != "Unbound" {
		t.Fatalf("text = %q", shortcutpage.KeyText(a, "close-tab"))
	}
}

// A key a text widget answers is refused and recording goes on; a key
// another action uses is not refused but offered with a Replace button (see
// shortcuteditor_test.go).
func TestRecordingRefusesKeysThatBelongToTextEditingAndKeepsListening(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyC, true, false, false))
	if !strings.Contains(a.shortcutKeys.Note("close-tab"), "text") || a.shortcutKeys.Recording() != "close-tab" {
		t.Fatalf("note %q recording %q", a.shortcutKeys.Note("close-tab"), a.shortcutKeys.Recording())
	}
	if len(a.settings.Shortcuts) != 0 {
		t.Fatal("refused combinations must change nothing")
	}
	a.step(combo(ebiten.KeyQ, true, false, false)) // a good one ends it and clears the note
	if a.shortcutKeys.Recording() != "" || a.shortcutKeys.Note("close-tab") != "" || a.settings.Shortcuts["close-tab"] != "Ctrl+Q" {
		t.Fatalf("recording %q note %q shortcuts %v", a.shortcutKeys.Recording(), a.shortcutKeys.Note("close-tab"), a.settings.Shortcuts)
	}
}

func TestAModifierKeyAloneDoesNotEndRecording(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyControlLeft}, Ctrl: true})
	if a.shortcutKeys.Recording() != "close-tab" || len(a.settings.Shortcuts) != 0 {
		t.Fatalf("recording %q shortcuts %v", a.shortcutKeys.Recording(), a.settings.Shortcuts)
	}
}

func TestRebindingToTheDefaultStoresNothingAndResetRestoresIt(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyQ, true, false, false))
	a.shortcutKeys.Reset(a, "close-tab")
	if _, ok := a.settings.Shortcuts["close-tab"]; ok || shortcutpage.KeyText(a, "close-tab") != "Ctrl+W" {
		t.Fatalf("shortcuts %v text %q", a.settings.Shortcuts, shortcutpage.KeyText(a, "close-tab"))
	}
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyW, true, false, false)) // its own default
	if _, ok := a.settings.Shortcuts["close-tab"]; ok {
		t.Fatalf("a binding equal to the default must not be stored: %v", a.settings.Shortcuts)
	}
}

func TestSavedBindingsApplyFromTheStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := settings.Default()
	s.Shortcuts = map[string]string{"new-request": "Ctrl+N", "teleport": "Ctrl+J"}
	if err := settingsDisk.Save(path, s); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(combo(ebiten.KeyN, true, false, false))
	if len(a.list.Tabs) != 2 {
		t.Fatal("the saved binding must work")
	}
	a.step(combo(ebiten.KeyT, true, false, false))
	if len(a.list.Tabs) != 2 {
		t.Fatal("the replaced default must not")
	}
}

func TestMenusShowTheConfiguredKey(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	a.shortcutKeys.Start("find")
	a.step(combo(ebiten.KeyG, true, false, false))
	rightClick(a, 1000, 400)
	if got := itemByLabel(t, a, "Find").Shortcut; got != "Ctrl+G" {
		t.Fatalf("the Find item shows %q", got)
	}
}

func TestShortcutSettingsRenderWhileRecording(t *testing.T) {
	a, _ := openSettingsPage(t, "shortcuts")
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyC, true, false, false)) // refused: it belongs to text editing, so a note shows
	a.step(core.Input{})                           // the menu with a recording row and a note must lay out
	if !a.menu || a.shortcutKeys.Recording() != "close-tab" {
		t.Fatal("the menu stays open while recording")
	}
}
