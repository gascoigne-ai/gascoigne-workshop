package core

import (
	"reflect"
	"testing"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
)

// A 24 px font has a line height near 27: the largest text size must not be
// cut down to a smaller line than its own face.
func TestSizesForTable(t *testing.T) {
	tests := []struct {
		name        string
		line, small int
		want        Sizes
	}{
		{"default", 16, 14, Sizes{16, 14, 28, 24, 20, 22, 18, 36, 40, 44, 40, 32, 44, 14}},
		{"small text", 13, 11, Sizes{13, 11, 25, 21, 17, 19, 15, 33, 37, 41, 37, 29, 41, 11}},
		{"line 15", 15, 13, Sizes{15, 13, 27, 23, 19, 21, 17, 35, 39, 43, 39, 31, 43, 13}},
		{"line 19", 19, 15, Sizes{19, 15, 31, 27, 23, 25, 19, 39, 43, 47, 43, 35, 47, 15}},
		{"a large line is not clamped", 27, 25, Sizes{27, 25, 39, 35, 31, 33, 29, 47, 51, 55, 51, 43, 55, 25}},
		{"line clamped high", 40, 20, Sizes{28, 20, 40, 36, 32, 34, 24, 48, 52, 56, 52, 44, 56, 20}},
		{"line clamped low, small clamped to line", 5, 4, Sizes{13, 4, 25, 21, 17, 19, 8, 33, 37, 41, 37, 29, 41, 4}},
		{"small clamped to line", 14, 30, Sizes{14, 14, 26, 22, 18, 20, 18, 34, 38, 42, 38, 30, 42, 14}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SizesFor(test.line, test.small)
			if got != test.want {
				t.Fatalf("SizesFor(%d, %d) = %+v, want %+v", test.line, test.small, got, test.want)
			}
			checkSizeInvariants(t, got)
		})
	}
}

func checkSizeInvariants(t *testing.T, sizes Sizes) {
	t.Helper()
	if sizes.Line < MinLine || sizes.Line > MaxLine {
		t.Errorf("Line %d outside %d..%d", sizes.Line, MinLine, MaxLine)
	}
	for name, height := range map[string]int{
		"Row": sizes.Row, "Compact": sizes.Compact, "Slim": sizes.Slim, "Label": sizes.Label,
		"Toolbar": sizes.Toolbar, "TabBar": sizes.TabBar, "DialogTitle": sizes.DialogTitle, "DialogToolbar": sizes.DialogToolbar,
	} {
		if height < sizes.Line || (height-sizes.Line)%2 != 0 {
			t.Errorf("%s = %d for line %d: want >= line and an even padding", name, height, sizes.Line)
		}
	}
	if sizes.Note < sizes.SmallLine || (sizes.Note-sizes.SmallLine)%2 != 0 {
		t.Errorf("Note = %d for small line %d", sizes.Note, sizes.SmallLine)
	}
	if !(sizes.Row > sizes.Compact && sizes.Compact > sizes.Slim) {
		t.Errorf("want Row > Compact > Slim, got %d %d %d", sizes.Row, sizes.Compact, sizes.Slim)
	}
}

// The constants stay as the default-face values until the later tasks that
// move every consumer to Ctx.Sizes delete them.
func TestDefaultSizesEqualConstants(t *testing.T) {
	want := Sizes{
		Line: BaseLine, SmallLine: BaseSmallLine, Row: RowH, Compact: CompactH, Slim: SlimH, Label: LabelH, Note: NoteH,
		Toolbar: ToolbarH, TabBar: TabBarH, DialogTitle: DialogTitleH, DialogToolbar: DialogToolbarH,
		IconBtnW: IconBtnW, WordBtnW: WordBtnW, Checkbox: CheckboxSize,
	}
	if got := DefaultSizes(); got != want {
		t.Fatalf("DefaultSizes() = %+v, want %+v", got, want)
	}
}

func TestSizesScopeNames(t *testing.T) {
	want := map[string]any{
		"rowH": 28, "compactH": 24, "slimH": 20, "labelH": 22, "noteH": 18, "toolbarH": 36, "tabBarH": 40,
		"dialogToolbarH": 40, "dialogTitleH": 44, "iconBtnW": 32, "wordBtnW": 44, "checkboxSize": 14,
		"tabLead": 2, "edgeW": 2, "nearGap": 4, "shadowOffset": 4, "tightGap": 6, "tabGap": 6,
		"ruleInset": 6, "edgeGrab": 6, "looseGap": 8, "cellPad": 8, "labelPad": 10, "tabSepInset": 10,
		"gutter": 12, "dialogPad": 16,
	}
	if got := DefaultSizes().Scope(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Scope() = %v, want %v", got, want)
	}

	big := SizesFor(20, 16).Scope()
	if big["rowH"] != 32 || big["gutter"] != 12 || big["tightGap"] != 6 {
		t.Fatalf("a larger line moves heights only: rowH %v gutter %v tightGap %v", big["rowH"], big["gutter"], big["tightGap"])
	}
}

func TestCtxStartsWithSizesOfItsFaces(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Sizes != DefaultSizes() {
		t.Fatalf("Sizes = %+v, want the default %+v", ctx.Sizes, DefaultSizes())
	}
}

func TestCtxSetThemeUpdatesSizes(t *testing.T) {
	ctx, _ := NewCtx(testTheme())
	large := testTheme()
	large.FontSize, large.SmallSize = 20, 16
	if err := ctx.SetTheme(large); err != nil {
		t.Fatal(err)
	}
	if ctx.Sizes.Line <= BaseLine || ctx.Sizes.Row <= RowH || ctx.Sizes.SmallLine <= BaseSmallLine {
		t.Fatalf("Sizes did not grow with the theme: %+v", ctx.Sizes)
	}
	want := SizesFor(ctx.LineH(ctx.F.UI), ctx.LineH(ctx.F.Small))
	if ctx.Sizes != want {
		t.Fatalf("Sizes = %+v, want %+v", ctx.Sizes, want)
	}
}

// The embedded Go fonts share their metrics, so no test font has another line
// height; the theme sizes are changed behind SetFonts' back instead, and the
// call must measure the faces again.
func TestCtxSetFontsUpdatesSizes(t *testing.T) {
	ctx, _ := NewCtx(testTheme())
	ctx.Th.FontSize, ctx.Th.SmallSize = 20, 16
	if err := ctx.SetFonts(FontData{Regular: gomono.TTF, Bold: gomonobold.TTF}, FontData{}); err != nil {
		t.Fatal(err)
	}
	want := SizesFor(ctx.LineH(ctx.F.UI), ctx.LineH(ctx.F.Small))
	if ctx.Sizes != want || ctx.Sizes == DefaultSizes() {
		t.Fatalf("Sizes = %+v, want %+v from the new faces", ctx.Sizes, want)
	}
	before := ctx.Sizes
	if err := ctx.SetFonts(FontData{Regular: []byte("nope")}, FontData{}); err == nil {
		t.Fatal("want an error for unreadable fonts")
	}
	if ctx.Sizes != before {
		t.Fatal("unreadable fonts changed Sizes")
	}
}

func TestCtxSetScaleKeepsSizes(t *testing.T) {
	for _, scale := range []float64{1.1, 1.5, 2} {
		ctx, _ := NewCtx(testTheme())
		before := ctx.Sizes
		if err := ctx.SetScale(scale); err != nil {
			t.Fatal(err)
		}
		if ctx.Sizes != before {
			t.Errorf("scale %v: Sizes = %+v, want %+v", scale, ctx.Sizes, before)
		}
	}
}
