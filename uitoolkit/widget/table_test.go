package widget

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestTableHeightAndEditHandlers(t *testing.T) {
	var toggled, deleted []int
	spec := TableSpec{
		Columns: []string{"Name", "Value"}, Weights: []float64{1, 1},
		Rows:        []TableRow{{Cells: []string{"q", "x"}, Checked: true}, {Cells: []string{"o", "y"}}},
		Placeholder: []string{"Name", "Value"},
		Editable:    true,
		OnToggle:    func(i int) { toggled = append(toggled, i) },
		OnDelete:    func(i int) { deleted = append(deleted, i) },
	}
	tbl := Table(spec)
	root := Col(tbl)
	c := testCtx(t)
	Layout(c, root, 400, 400)

	// header + 2 rows + placeholder
	if got, want := tbl.Rect().H, 4*core.RowH; got != want {
		t.Fatalf("table height = %d, want %d", got, want)
	}
	rowY := func(i int) int { return (i+1)*core.RowH + core.RowH/2 }
	click(c, 10, rowY(1))
	Update(c, root)
	click(c, 395, rowY(0))
	Update(c, root)
	click(c, 10, rowY(2)) // placeholder row has no actions
	Update(c, root)

	if len(toggled) != 1 || toggled[0] != 1 || len(deleted) != 1 || deleted[0] != 0 {
		t.Fatalf("toggled %v deleted %v; want [1] and [0]", toggled, deleted)
	}
}

// Columns of a 400px editable table: 36px actions, then two 164px cells.
const cellX0, cellX1 = 36 + 80, 36 + 164 + 80

func editableSpec() TableSpec {
	return TableSpec{
		Columns: []string{"Name", "Value"}, Weights: []float64{1, 1},
		Rows:        []TableRow{{Cells: []string{"q", "x"}}, {Cells: []string{"o", "y"}}},
		Placeholder: []string{"Name", "Value"},
		Editable:    true,
		OnToggle:    func(int) {}, OnDelete: func(int) {},
	}
}

func cellY(row int) int { return (row+1)*core.RowH + core.RowH/2 }

func TestClickingACellStartsEditingIt(t *testing.T) {
	var got [][2]int
	spec := editableSpec()
	spec.OnEditCell = func(row, col int) { got = append(got, [2]int{row, col}) }
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)

	click(c, cellX1, cellY(1))
	Update(c, root)
	click(c, cellX0, cellY(2)) // the placeholder row is editable too
	Update(c, root)

	if len(got) != 2 || got[0] != [2]int{1, 1} || got[1] != [2]int{2, 0} {
		t.Fatalf("edit requests = %v, want [[1 1] [2 0]]", got)
	}
}

func TestReadOnlyTableCellsIgnoreClicks(t *testing.T) {
	called := false
	spec := editableSpec()
	spec.Editable = false
	spec.OnEditCell = func(int, int) { called = true }
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)
	click(c, cellX0-36, cellY(0)) // read-only tables have no action column: shift left
	Update(c, root)
	if called {
		t.Fatal("a read-only table must not start edits")
	}
}

func TestEditedCellShowsAFieldWithEnterAndTab(t *testing.T) {
	var enters int
	var tabs []bool
	f := NewTextField("x")
	f.Focused = true
	spec := editableSpec()
	spec.Editing = &TableEdit{Row: 0, Col: 1, Field: &f,
		OnEnter: func() { enters++ }, OnTab: func(back bool) { tabs = append(tabs, back) }}
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)

	c.In = core.Input{Typed: []rune("yz")}
	Update(c, root)
	if f.Text != "xyz" {
		t.Fatalf("typing reached the field as %q, want xyz", f.Text)
	}
	c.In = core.Input{Enter: true}
	Update(c, root)
	c.In = core.Input{Tab: true}
	Update(c, root)
	c.In = core.Input{Tab: true, Shift: true}
	Update(c, root)
	if enters != 1 || len(tabs) != 2 || tabs[0] || !tabs[1] {
		t.Fatalf("enters %d tabs %v, want 1 and [false true]", enters, tabs)
	}
}

func TestTableRowRightClickReportsTheRowButNotThePlaceholder(t *testing.T) {
	var rows []int
	spec := editableSpec()
	spec.OnRowMenu = func(row, x, y int) { rows = append(rows, row) }
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)
	for _, row := range []int{0, 1, 2} { // 2 is the placeholder row
		c.In = core.Input{X: 150, Y: cellY(row), RightClick: true}
		Update(c, root)
	}
	if !reflect.DeepEqual(rows, []int{0, 1}) {
		t.Fatalf("rows = %v", rows)
	}
}

func TestTableCellStylerSplitsACellIntoColoredParts(t *testing.T) {
	spec := TableSpec{
		Columns: []string{"Name", "Value"}, Weights: []float64{1, 1},
		Rows: []TableRow{{Cells: []string{"k", "a{{v}}b"}}},
		CellStyler: func(text string) []Span {
			if text != "a{{v}}b" {
				return nil
			}
			return []Span{{Text: "a"}, {Text: "{{v}}", Style: core.OK}, {Text: "b"}}
		},
	}
	d := Dump(Table(spec))
	for _, want := range []string{`text="a"`, `text="{{v}}"`, `fg=ok`, `text="b"`} {
		if !strings.Contains(d, want) {
			t.Errorf("missing %s in\n%s", want, d)
		}
	}
	if !strings.Contains(d, `text="k"`) {
		t.Errorf("a cell the styler declines stays plain:\n%s", d)
	}
}
