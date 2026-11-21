package settingsui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestFontSizeControlSteps(t *testing.T) {
	cases := []struct {
		name   string
		start  int
		button string
		want   int
	}{
		{"plus steps up", 14, `text="+"`, 15},
		{"minus steps down", 14, `text="-"`, 13},
		{"plus at the top does nothing", settings.MaxFontSize, `text="+"`, settings.MaxFontSize},
		{"minus at the bottom does nothing", settings.MinFontSize, `text="-"`, settings.MinFontSize},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.host.current.FontSize = tc.start
			changes := driver.host.changes
			driver.clickOn(0, tc.button)
			if driver.host.current.FontSize != tc.want {
				t.Fatalf("font size %d after the click, want %d", driver.host.current.FontSize, tc.want)
			}
			if tc.start == tc.want && driver.host.changes != changes {
				t.Fatal("a step at the end of the range changed a setting")
			}
			if got := driver.dialog.Input(driver.host, "fontSize").Text; got != fmt.Sprint(tc.want) {
				t.Fatalf("field shows %q, want %d", got, tc.want)
			}
		})
	}
}

func TestFontSizeFieldAppliesTypedNumbersOrKeepsTheLastGoodOne(t *testing.T) {
	cases := []struct {
		name  string
		typed string
		want  int
		note  string
	}{
		{"in range", "18", 18, ""},
		{"too big", "30", 14, "Enter a whole number between 10 and 24"},
		{"too small", "9", 14, "Enter a whole number between 10 and 24"},
		{"not a number", "big", 14, "Enter a whole number between 10 and 24"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.clickOn(0, "fieldB") // the text size is the first field of the page
			driver.retype(tc.typed)
			if driver.host.current.FontSize != tc.want || driver.dialog.Note("fontSize") != tc.note {
				t.Fatalf("font size %d note %q, want %d %q", driver.host.current.FontSize, driver.dialog.Note("fontSize"), tc.want, tc.note)
			}
			if tc.note != "" && !strings.Contains(driver.view(), `text="`+tc.note+`"`) {
				t.Fatalf("the note is not under the field:\n%s", driver.view())
			}
		})
	}
}

// A step after a refused entry goes from the setting, not from the text.
func TestFontSizeStepAfterARefusedEntryStartsFromTheSetting(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.clickOn(0, "fieldB")
	driver.retype("30")
	driver.clickOn(0, `text="+"`)
	if driver.host.current.FontSize != 15 || driver.dialog.Note("fontSize") != "" {
		t.Fatalf("font size %d note %q, want 15 and no note", driver.host.current.FontSize, driver.dialog.Note("fontSize"))
	}
}

func TestResetPutsTheDefaultFontSizeBackIntoTheField(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.host.current.FontSize = 20
	driver.step(func(*El) core.Input { return core.Input{} })
	driver.clickOn(0, `text="Reset"`)
	driver.step(func(*El) core.Input { return core.Input{} })
	if driver.host.current.FontSize != settings.DefaultFontSize {
		t.Fatalf("font size %d after Reset", driver.host.current.FontSize)
	}
	if got := driver.dialog.Input(driver.host, "fontSize").Text; got != "14" {
		t.Fatalf("field shows %q after Reset, want 14", got)
	}
}

func TestSettingControlsHaveNoTextSizeChips(t *testing.T) {
	if _, ok := settingControls["textSize"]; ok {
		t.Fatal("textSize still has a control")
	}
	page := newFormDriver(t, "Appearance").view()
	for _, label := range []string{"Small", "Medium", "Large"} {
		if strings.Contains(page, `text="`+label+`"`) {
			t.Errorf("a %s chip is still on the page", label)
		}
	}
	if !strings.Contains(page, `text="Text size"`) {
		t.Error("the Text size row is gone")
	}
}

// ctxAtFont is a context whose theme has the sizes of a text size setting.
func ctxAtFont(t *testing.T, font int) *core.Ctx {
	t.Helper()
	theme := testTheme()
	theme.FontSize, theme.SmallSize, theme.MonoSize = theming.TextSizesFor(font)
	ctx, err := core.NewCtx(theme)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// controlsFit fails for a control shorter than its text's line and for two
// siblings in a flow that overlap.
func controlsFit(t *testing.T, ctx *core.Ctx, el *El) {
	t.Helper()
	line, _, _ := strings.Cut(Dump(el), "\n")
	isControl := strings.Contains(line, "control=") || strings.HasPrefix(strings.TrimSpace(line), "fieldB")
	if rect := el.Rect(); isControl && rect.H < ctx.Sizes.Line {
		t.Errorf("control %q is %d px high, its line is %d", line, rect.H, ctx.Sizes.Line)
	}
	var placed []*El
	for _, kid := range el.Children() {
		if kid.InFlow() {
			placed = append(placed, kid)
		}
	}
	for i, first := range placed {
		for _, second := range placed[i+1:] {
			if overlaps(first.Rect(), second.Rect()) {
				a, _, _ := strings.Cut(Dump(first), "\n")
				b, _, _ := strings.Cut(Dump(second), "\n")
				t.Errorf("siblings overlap:\n%s\n%s", a, b)
			}
		}
	}
	for _, kid := range el.Children() {
		controlsFit(t, ctx, kid)
	}
}

func overlaps(a, b core.Rect) bool {
	if a.W == 0 || a.H == 0 || b.W == 0 || b.H == 0 {
		return false
	}
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// The ends of the text size range are where a fixed height would clip.
func TestControlsFitAtFontRangeEnds(t *testing.T) {
	for _, font := range []int{settings.MinFontSize, settings.DefaultFontSize, settings.MaxFontSize} {
		for _, category := range settings.Categories {
			t.Run(fmt.Sprintf("%d/%s", font, category), func(t *testing.T) {
				ctx := ctxAtFont(t, font)
				var dialog SettingsDialog
				dialog.Families = testFamilies(t)
				host := newHost(t)
				host.current.FontSize = font
				host.current.CACert = "/etc/ssl/ca.pem"
				root := dialog.FormPage(ctx, host, category)
				Layout(ctx, root, formWideW, formWideH)
				controlsFit(t, ctx, root)
			})
		}
	}
}
