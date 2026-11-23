package app

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var rectPattern = regexp.MustCompile(`rect=(-?\d+),(-?\d+),(\d+),(\d+)`)

// layoutDump is the window as text with every element's rectangle.
func layoutDump(app *App) string {
	root := app.view()
	view.Layout(app.ctx, root, app.w, app.h)
	return view.Dump(root)
}

// clickText clicks the middle of the nth (from 0) element that shows text
// exactly, so a test finds a control by its label and not by pixel.
func clickText(t *testing.T, app *App, text string, nth int) {
	t.Helper()
	clickDumped(t, app, fmt.Sprintf("text=%q", text), nth)
}

func settingsText(app *App) string { return view.Dump(app.prefs.View(app.ctx)) }

func TestTheSettingsFormShowsATitleADescriptionAndAControlPerSetting(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	dump := settingsText(a)
	for _, want := range []string{"Request and response", "Place the request and response panes", "Side by side", "Stacked", "Interface scale", "Flat colors"} {
		if !strings.Contains(dump, want) {
			t.Errorf("the Appearance page lacks %q", want)
		}
	}
}

func TestTheRequestsPageHoldsTimeoutProxyAndCertificates(t *testing.T) {
	a, _ := openSettingsPage(t, "requests")
	dump := settingsText(a)
	for _, want := range []string{"Request timeout", "Proxy", "CA certificates", "Client certificate", "Client key"} {
		if !strings.Contains(dump, want) {
			t.Errorf("the Requests page lacks %q", want)
		}
	}
	if strings.Contains(dump, "Collection folder") || strings.Contains(dump, "Active environment") {
		t.Error("internal state is shown in the form")
	}
}

func TestSearchFiltersSettingsAcrossCategories(t *testing.T) {
	tests := []struct {
		query   string
		want    []string
		notWant []string
	}{
		{"proxy", []string{"Proxy"}, []string{`text="Corners"`, "Request and response"}},
		{"CORNERS", []string{"Corners"}, []string{"Proxy"}},
		{"gradient", []string{"Flat colors"}, []string{"Proxy", `text="Corners"`}},         // matches the description
		{"stacked", []string{"Request and response"}, []string{"Proxy", `text="Corners"`}}, // matches an allowed value
		{"timeoutSec", []string{"Request timeout"}, []string{`text="Corners"`}},            // matches the key
		{"zzzz", []string{"No settings match"}, []string{"Proxy", `text="Corners"`}},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			a, _ := openSettingsPage(t, "appearance")
			a.prefs.Shell.Search = widget.NewTextField(test.query)
			dump := settingsText(a)
			for _, want := range test.want {
				if !strings.Contains(dump, want) {
					t.Errorf("results lack %q", want)
				}
			}
			for _, bad := range test.notWant {
				if strings.Contains(dump, bad) {
					t.Errorf("results contain %q", bad)
				}
			}
		})
	}
}

func TestAChangedSettingIsMarkedAndResetRemovesItFromTheFile(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.prefs.Shell.Search = widget.NewTextField("request and response") // brings the setting to the top of the page
	clickText(t, a, "Side by side", 0)
	if a.settings.Layout != "side" {
		t.Fatalf("Layout = %q", a.settings.Layout)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), `"layout": "side"`) {
		t.Fatalf("file = %s", data)
	}
	if !strings.Contains(settingsText(a), "icon=undo") {
		t.Fatal("a changed setting has no reset icon")
	}
	clickDumped(t, a, "icon=undo", 0)
	if a.settings.Layout != "stacked" {
		t.Fatalf("Layout = %q after reset", a.settings.Layout)
	}
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "layout") {
		t.Fatalf("the reset key is still in the file: %s", data)
	}
	if strings.Contains(settingsText(a), "icon=undo") {
		t.Fatal("the reset icon is shown for a setting at its default")
	}
}

func TestTextSettingsApplyAsTheyAreTyped(t *testing.T) {
	a, path := openSettingsPage(t, "requests")
	input := a.prefs.Form.Input(a.prefs, "proxy")
	*input = widget.NewTextField("http://localhost:8080")
	a.prefs.Form.Edited(a.prefs, "proxy")
	if a.settings.Proxy != "http://localhost:8080" {
		t.Fatalf("Proxy = %q", a.settings.Proxy)
	}
	if saved, _ := settingsDisk.Load(path); saved.Proxy != "http://localhost:8080" {
		t.Fatalf("saved %+v", saved)
	}
}

func TestANumberSettingNeedsAWholeNumberInRange(t *testing.T) {
	a, _ := openSettingsPage(t, "requests")
	input := a.prefs.Form.Input(a.prefs, "timeoutSec")
	for _, test := range []struct {
		text     string
		want     int
		wantNote bool
	}{
		{"30", 30, false},
		{"abc", 30, true},
		{"0", 30, true},
		{"99999", 30, true},
		{"90", 90, false},
	} {
		*input = widget.NewTextField(test.text)
		a.prefs.Form.Edited(a.prefs, "timeoutSec")
		if a.settings.TimeoutSec != test.want || (a.prefs.Form.Note("timeoutSec") != "") != test.wantNote {
			t.Errorf("%q: TimeoutSec = %d (want %d), note %q", test.text, a.settings.TimeoutSec, test.want, a.prefs.Form.Note("timeoutSec"))
		}
	}
}

func TestClosingTheSettingsAlwaysOpensThemFresh(t *testing.T) {
	a, _ := openSettingsPage(t, "requests")
	a.prefs.Shell.Search = widget.NewTextField("proxy")
	a.prefs.OpenRaw()
	*a.prefs.Form.Input(a.prefs, "proxy") = widget.NewTextField("typed but not entered")
	a.prefs.Close()
	a.prefs.Show()
	if a.prefs.Shell.Cat != "" || a.prefs.Shell.Search.Text != "" || a.prefs.Shell.Raw || a.prefs.Form.FontListOpen() {
		t.Fatalf("page %q search %q raw %v: the reopened dialog is not fresh", a.prefs.Shell.Cat, a.prefs.Shell.Search.Text, a.prefs.Shell.Raw)
	}
	if got := a.prefs.Form.Input(a.prefs, "proxy").Text; got != a.settings.Proxy {
		t.Fatalf("a field kept text typed before the dialog was closed: %q", got)
	}
	if a.prefs.Shell.Searching() {
		t.Fatal("a search is still active")
	}
}
