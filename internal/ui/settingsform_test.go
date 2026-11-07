package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

var rectPattern = regexp.MustCompile(`rect=(-?\d+),(-?\d+),(\d+),(\d+)`)

// layoutDump is the window as text with every element's rectangle.
func layoutDump(a *App) string {
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	return view.Dump(root)
}

// clickText clicks the middle of the nth (from 0) element that shows text
// exactly, so a test finds a control by its label and not by pixel.
func clickText(t *testing.T, a *App, text string, nth int) {
	t.Helper()
	a.step(core.Input{X: 5000, Y: 5000})
	needle := fmt.Sprintf("text=%q", text)
	seen := 0
	for _, line := range strings.Split(layoutDump(a), "\n") {
		if !strings.Contains(line, needle) {
			continue
		}
		if seen++; seen-1 != nth {
			continue
		}
		m := rectPattern.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("no rectangle on %q", line)
		}
		var x, y, w, h int
		fmt.Sscan(m[1], &x)
		fmt.Sscan(m[2], &y)
		fmt.Sscan(m[3], &w)
		fmt.Sscan(m[4], &h)
		click(a, x+w/2, y+h/2)
		return
	}
	t.Fatalf("no element with text %q (#%d) in:\n%s", text, nth, layoutDump(a))
}

func settingsText(a *App) string { return view.Dump(a.menuView()) }

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
		{"proxy", []string{"Proxy"}, []string{"Corners", "Request and response"}},
		{"CORNERS", []string{"Corners"}, []string{"Proxy"}},
		{"gradient", []string{"Flat colors"}, []string{"Proxy", "Corners"}},         // matches the description
		{"stacked", []string{"Request and response"}, []string{"Proxy", "Corners"}}, // matches an allowed value
		{"timeoutSec", []string{"Request timeout"}, []string{"Corners"}},            // matches the key
		{"zzzz", []string{"No settings match"}, []string{"Proxy", "Corners"}},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			a, _ := openSettingsPage(t, "appearance")
			a.settingsSearch = view.NewTextField(test.query)
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
	a.settingsSearch = view.NewTextField("request and response") // brings the setting to the top of the page
	clickText(t, a, "Stacked", 0)
	if a.settings.Layout != "stacked" {
		t.Fatalf("Layout = %q", a.settings.Layout)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), `"layout": "stacked"`) {
		t.Fatalf("file = %s", data)
	}
	if !strings.Contains(settingsText(a), "Reset") {
		t.Fatal("a changed setting has no Reset button")
	}
	clickText(t, a, "Reset", 0)
	if a.settings.Layout != "side" {
		t.Fatalf("Layout = %q after reset", a.settings.Layout)
	}
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "layout") {
		t.Fatalf("the reset key is still in the file: %s", data)
	}
	if strings.Contains(settingsText(a), `text="Reset"`) {
		t.Fatal("Reset is shown for a setting at its default")
	}
}

func TestTextSettingsApplyAsTheyAreTyped(t *testing.T) {
	a, path := openSettingsPage(t, "requests")
	input := a.settingInput("proxy")
	*input = view.NewTextField("http://localhost:8080")
	a.settingEdited("proxy")
	if a.settings.Proxy != "http://localhost:8080" {
		t.Fatalf("Proxy = %q", a.settings.Proxy)
	}
	if saved, _ := settings.Load(path); saved.Proxy != "http://localhost:8080" {
		t.Fatalf("saved %+v", saved)
	}
}

func TestANumberSettingNeedsAWholeNumberInRange(t *testing.T) {
	a, _ := openSettingsPage(t, "requests")
	input := a.settingInput("timeoutSec")
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
		*input = view.NewTextField(test.text)
		a.settingEdited("timeoutSec")
		if a.settings.TimeoutSec != test.want || (a.settingNotes["timeoutSec"] != "") != test.wantNote {
			t.Errorf("%q: TimeoutSec = %d (want %d), note %q", test.text, a.settings.TimeoutSec, test.want, a.settingNotes["timeoutSec"])
		}
	}
}
