package settingsui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// schemaField is the row of key in the schema.
func schemaField(t *testing.T, key string) settings.Field {
	t.Helper()
	for _, field := range settings.Schema {
		if field.Key == key {
			return field
		}
	}
	t.Fatalf("no setting %q in the schema", key)
	return settings.Field{}
}

// faceTitles are the titles of the font of each face.
var faceTitles = map[string]string{"uiFont": "Interface font", "codeFont": "Code font"}

// rowOf is the row of the page that holds the title.
func rowOf(t *testing.T, root *El, title string) *El {
	t.Helper()
	for _, row := range root.Children() {
		if len(findAll(row, `text="`+title+`"`, "bold=true")) > 0 {
			return row
		}
	}
	t.Fatalf("no row titled %q:\n%s", title, Dump(root))
	return nil
}

func TestEveryFontFaceNamesRowsOfTheSchema(t *testing.T) {
	for _, face := range fontFaces {
		font, size := schemaField(t, face.Font), schemaField(t, face.Size)
		if font.Category != size.Category || size.Kind != settings.Int || font.Hidden {
			t.Errorf("face %v: font %+v and size %+v do not pair", face, font, size)
		}
	}
	if len(fontFaces) != 2 {
		t.Errorf("%d faces, want the interface and the code face", len(fontFaces))
	}
}

// The sizes of the two faces are not rows of their own; Small text size, which
// has no font, stays a plain row under the interface line.
func TestFaceSizesAreNotRowsOfTheirOwn(t *testing.T) {
	page := newFormDriver(t, "Appearance").view()
	for _, title := range []string{"Interface font size", "Code font size"} {
		if strings.Contains(page, `text="`+title+`"`) {
			t.Errorf("%q is still a row of its own", title)
		}
	}
	last := -1
	for _, title := range []string{"Interface font", "Small text size", "Code font", "Interface scale"} {
		at := strings.Index(page, `text="`+title+`"`)
		if at < 0 || at < last {
			t.Fatalf("row %q is missing or out of order:\n%s", title, page)
		}
		last = at
	}
}

// faceLine is the font drop-down, the stepper (its minus button and the
// field) and the size Reset icon slot of a face row.
type faceLine struct{ font, minus, field, plus *El }

func faceLineOf(t *testing.T, row *El) faceLine {
	t.Helper()
	return faceLine{
		font:  firstOf(t, row, "tip=Pick an installed font"),
		minus: firstOf(t, row, "tip=Smaller"),
		field: firstOf(t, row, "fieldB"),
		plus:  firstOf(t, row, "tip=Larger"),
	}
}

// The font and the stepper of a face share one line, side by side, inside the
// row, and the font keeps its minimum width.
func TestAFontAndItsSizeShareOneLine(t *testing.T) {
	for _, width := range []int{DialogWidth - DialogNavW, 560, rowWidths[1]} {
		for _, title := range faceTitles {
			t.Run(fmt.Sprintf("%s/%d", title, width), func(t *testing.T) {
				driver := newFormDriver(t, "Appearance")
				driver.host.current.UIFont = "A family name that is long and not installed anywhere"
				driver.host.current.CodeFont = driver.host.current.UIFont
				driver.host.current.FontSize = settings.DefaultFontSize + 1 // the size Reset shows
				driver.host.current.CodeFontSize = settings.DefaultCodeFontSize + 1
				root := rowPage(t, driver, width)
				row := rowOf(t, root, title)
				line := faceLineOf(t, row)
				font, minus, field, plus := line.font.Rect(), line.minus.Rect(), line.field.Rect(), line.plus.Rect()
				if font.H != field.H {
					t.Errorf("font %v and size field %v are not on one line", font, field)
				}
				if font.Y >= field.Y+field.H || field.Y >= font.Y+font.H {
					t.Errorf("font %v and size field %v do not share a y range", font, field)
				}
				if font.X+font.W > minus.X || minus.X+minus.W > field.X || field.X+field.W > plus.X {
					t.Errorf("font %v, minus %v, field %v, plus %v overlap or are out of order", font, minus, field, plus)
				}
				if font.W < fontMinW {
					t.Errorf("font is %d wide, minimum is %d", font.W, fontMinW)
				}
				for _, part := range []core.Rect{font, minus, field, plus} {
					if !within(part, row.Rect()) {
						t.Errorf("%v leaves its row %v", part, row.Rect())
					}
				}
				for _, reset := range findAll(row, "icon=undo") {
					if !within(reset.Rect(), row.Rect()) || overlaps(reset.Rect(), plus) || overlaps(reset.Rect(), font) {
						t.Errorf("reset %v overlaps or leaves the row %v", reset.Rect(), row.Rect())
					}
				}
			})
		}
	}
}

// The size Reset slot is reserved while the size is unchanged, so the steppers
// of both faces and the width of the font do not move when it shows.
func TestTheSizeResetSlotKeepsTheLineStable(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	before := faceLineOf(t, rowOf(t, rowPage(t, driver, 560), "Interface font"))
	driver.host.current.FontSize = settings.DefaultFontSize + 1
	after := faceLineOf(t, rowOf(t, rowPage(t, driver, 560), "Interface font"))
	if before.font.Rect() != after.font.Rect() || before.field.Rect() != after.field.Rect() {
		t.Errorf("the line moved when the size changed: font %v to %v, field %v to %v",
			before.font.Rect(), after.font.Rect(), before.field.Rect(), after.field.Rect())
	}
}

// A search for "size" lists each face line once, and Small text size, with
// one caption for the category.
func TestSearchFindsAFaceLineOnceThroughEitherSetting(t *testing.T) {
	cases := []struct {
		query string
		want  []string
	}{
		{"size", []string{"Interface font", "Small text size", "Code font"}},
		{"fontSize", []string{"Interface font"}},
		{"uiFont", []string{"Interface font"}},
		{"codeFontSize", []string{"Code font"}},
		{"codeFont", []string{"Code font"}},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.dialog.Search = NewTextField(tc.query)
			root := rowPage(t, driver, 560)
			for _, title := range tc.want {
				if n := len(findAll(root, `text="`+title+`"`, "bold=true")); n != 1 {
					t.Errorf("%q: %d rows titled %q:\n%s", tc.query, n, title, Dump(root))
				}
			}
			if n := len(findAll(root, `text="APPEARANCE"`)); n != 1 {
				t.Errorf("%q: %d Appearance captions, want 1", tc.query, n)
			}
			if strings.HasPrefix(tc.query, "code") || tc.query == "uiFont" {
				if len(findAll(root, `text="Small text size"`)) > 0 {
					t.Errorf("%q: Small text size is listed", tc.query)
				}
			}
		})
	}
}

// The two Resets of a face are independent: the font's puts the font back and
// applies the fonts, the size's leaves the font alone.
func TestTheFontAndSizeResetsOfAFaceAreIndependent(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.host.current.UIFont = "Go"
	driver.host.current.FontSize = 16
	resets := findAll(rowOf(t, rowPage(t, driver, 560), "Interface font"), "icon=undo")
	if len(resets) != 2 {
		t.Fatalf("%d reset icons in the face row, want font and size", len(resets))
	}
	driver.clickOn(0, "icon=undo") // the one by the title
	if driver.host.current.UIFont != "" || driver.host.current.FontSize != 16 {
		t.Fatalf("font %q size %d after the font Reset, want \"\" and 16", driver.host.current.UIFont, driver.host.current.FontSize)
	}
	if driver.host.fontsSet == 0 {
		t.Error("the font Reset did not apply the fonts")
	}
	driver.host.current.UIFont = "Go"
	driver.clickOn(1, "icon=undo") // the one by the stepper
	if driver.host.current.UIFont != "Go" || driver.host.current.FontSize != settings.DefaultFontSize {
		t.Fatalf("font %q size %d after the size Reset, want Go and %d", driver.host.current.UIFont, driver.host.current.FontSize, settings.DefaultFontSize)
	}
}

// A refused size shows its note under the line, inside the row.
func TestARefusedFaceSizeShowsItsNoteInsideTheRow(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.clickOn(0, "fieldB")
	driver.retype("99")
	root := rowPage(t, driver, 560)
	row := rowOf(t, root, "Interface font")
	note := firstOf(t, row, `text="`+rangeNote(faceSizeRows[0])+`"`)
	if !within(note.Rect(), row.Rect()) {
		t.Errorf("note %v leaves its row %v", note.Rect(), row.Rect())
	}
	if driver.host.current.FontSize != settings.DefaultFontSize {
		t.Errorf("size %d, want the last good one", driver.host.current.FontSize)
	}
}

// The description of a face holds the range of its size once, taken from the
// schema row.
func TestAFaceDescriptionStatesTheSizeRangeFromTheSchema(t *testing.T) {
	root := rowPage(t, newFormDriver(t, "Appearance"), 560)
	for _, face := range fontFaces {
		size := schemaField(t, face.Size)
		want := fmt.Sprintf("%d to %d", size.Min, size.Max)
		paragraphs := findAll(rowOf(t, root, faceTitles[face.Font]), "paragraphB")
		if len(paragraphs) != 1 || strings.Count(Dump(paragraphs[0]), want) != 1 {
			t.Errorf("face %v: want one description with %q once:\n%s", face, want, Dump(rowOf(t, root, faceTitles[face.Font])))
		}
	}
}
