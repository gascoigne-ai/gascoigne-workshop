package settingsui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// faceSizeRow is one of the three size rows of the Appearance page: each is
// a number field with a stepper, in the order of the page.
type faceSizeRow struct {
	key        string
	title      string
	min, max   int
	field      int // which fieldB of the page it is
	value      func(*settings.Settings) *int
	defaultVal int
}

var faceSizeRows = []faceSizeRow{
	{"fontSize", "Interface font size", settings.MinFontSize, settings.MaxFontSize, 0, func(prefs *settings.Settings) *int { return &prefs.FontSize }, settings.DefaultFontSize},
	{"smallFontSize", "Small text size", settings.MinSmallFontSize, settings.MaxSmallFontSize, 1, func(prefs *settings.Settings) *int { return &prefs.SmallFontSize }, settings.DefaultSmallFontSize},
	{"codeFontSize", "Code font size", settings.MinCodeFontSize, settings.MaxCodeFontSize, 2, func(prefs *settings.Settings) *int { return &prefs.CodeFontSize }, settings.DefaultCodeFontSize},
}

func rangeNote(row faceSizeRow) string {
	return fmt.Sprintf("Enter a whole number between %d and %d", row.min, row.max)
}

func TestFaceSizeControlSteps(t *testing.T) {
	for _, row := range faceSizeRows {
		cases := []struct {
			name   string
			start  int
			button string
			want   int
		}{
			{"plus steps up", row.defaultVal, `text="+"`, row.defaultVal + 1},
			{"minus steps down", row.defaultVal, `text="-"`, row.defaultVal - 1},
			{"plus at the top does nothing", row.max, `text="+"`, row.max},
			{"minus at the bottom does nothing", row.min, `text="-"`, row.min},
		}
		for _, tc := range cases {
			t.Run(row.key+"/"+tc.name, func(t *testing.T) {
				driver := newFormDriver(t, "Appearance")
				*row.value(&driver.host.current) = tc.start
				changes := driver.host.changes
				driver.clickOn(row.field, tc.button)
				if got := *row.value(&driver.host.current); got != tc.want {
					t.Fatalf("%s %d after the click, want %d", row.key, got, tc.want)
				}
				if tc.start == tc.want && driver.host.changes != changes {
					t.Fatal("a step at the end of the range changed a setting")
				}
				if got := driver.dialog.Input(driver.host, row.key).Text; got != fmt.Sprint(tc.want) {
					t.Fatalf("field shows %q, want %d", got, tc.want)
				}
			})
		}
	}
}

func TestFaceSizeFieldAppliesTypedNumbersOrKeepsTheLastGoodOne(t *testing.T) {
	for _, row := range faceSizeRows {
		cases := []struct {
			name  string
			typed string
			want  int
			note  string
		}{
			{"in range", fmt.Sprint(row.min + 1), row.min + 1, ""},
			{"too big", fmt.Sprint(row.max + 1), row.defaultVal, rangeNote(row)},
			{"too small", fmt.Sprint(row.min - 1), row.defaultVal, rangeNote(row)},
			{"not a number", "big", row.defaultVal, rangeNote(row)},
		}
		for _, tc := range cases {
			t.Run(row.key+"/"+tc.name, func(t *testing.T) {
				driver := newFormDriver(t, "Appearance")
				driver.clickOn(row.field, "fieldB")
				driver.retype(tc.typed)
				if got := *row.value(&driver.host.current); got != tc.want || driver.dialog.Note(row.key) != tc.note {
					t.Fatalf("%s %d note %q, want %d %q", row.key, got, driver.dialog.Note(row.key), tc.want, tc.note)
				}
				if tc.note != "" && !strings.Contains(driver.view(), `text="`+tc.note+`"`) {
					t.Fatalf("the note is not under the field:\n%s", driver.view())
				}
			})
		}
	}
}

// A step after a refused entry goes from the setting, not from the text.
func TestFaceSizeStepAfterARefusedEntryStartsFromTheSetting(t *testing.T) {
	for _, row := range faceSizeRows {
		t.Run(row.key, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.clickOn(row.field, "fieldB")
			driver.retype(fmt.Sprint(row.max + 1))
			driver.clickOn(row.field, `text="+"`)
			if got := *row.value(&driver.host.current); got != row.defaultVal+1 || driver.dialog.Note(row.key) != "" {
				t.Fatalf("%s %d note %q, want %d and no note", row.key, got, driver.dialog.Note(row.key), row.defaultVal+1)
			}
		})
	}
}

func TestResetPutsTheDefaultFaceSizeBackIntoTheField(t *testing.T) {
	for _, row := range faceSizeRows {
		t.Run(row.key, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			*row.value(&driver.host.current) = row.max
			driver.step(func(*El) core.Input { return core.Input{} })
			driver.clickOn(0, "icon=undo") // the only changed row
			driver.step(func(*El) core.Input { return core.Input{} })
			if got := *row.value(&driver.host.current); got != row.defaultVal {
				t.Fatalf("%s %d after Reset", row.key, got)
			}
			if got := driver.dialog.Input(driver.host, row.key).Text; got != fmt.Sprint(row.defaultVal) {
				t.Fatalf("field shows %q after Reset, want %d", got, row.defaultVal)
			}
		})
	}
}

// Each size sits on the line of the font it sizes; Small text size follows the
// interface line, the scale comes after both.
func TestSizeRowsSitNextToTheirFonts(t *testing.T) {
	page := newFormDriver(t, "Appearance").view()
	last := -1
	for _, title := range []string{"Interface font", "Small text size", "Code font", "Interface scale"} {
		at := strings.Index(page, `text="`+title+`"`)
		if at < 0 || at < last {
			t.Fatalf("row %q is missing or out of order in the page:\n%s", title, page)
		}
		last = at
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
	if strings.Contains(page, `text="Text size"`) {
		t.Error("the old Text size title is still on the page")
	}
}

// ctxAtFont is a context whose theme has the sizes of a text size setting.
func ctxAtFont(t *testing.T, font int) *core.Ctx {
	t.Helper()
	theme := testTheme()
	theme = theme.WithTextSize(theming.FontSizes{UI: font, Small: max(font-2, 9), Code: font - 1}.Resolved())
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
