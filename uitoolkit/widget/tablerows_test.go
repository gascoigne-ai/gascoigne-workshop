package widget

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func readOnlySpec() TableSpec {
	return TableSpec{
		Columns: []string{"Name", "Value"}, Weights: []float64{1, 1},
		Rows: []TableRow{{Cells: []string{"a", "1"}}, {Cells: []string{"b", "2"}}, {Cells: []string{"c", "3"}}, {Cells: []string{"d", "4"}}},
	}
}

// rowsOf are the data rows of a laid-out table: the children after the header.
func rowsOf(table *El) []*El { return table.Children()[1:] }

func TestZebraFillsEverySecondRowWithAlt(t *testing.T) {
	spec := readOnlySpec()
	spec.Zebra = true
	rows := rowsOf(Table(spec))
	for index, row := range rows {
		first, _, _ := strings.Cut(Dump(row), "\n")
		if got, want := strings.Contains(first, "bg=alt"), index%2 == 1; got != want {
			t.Errorf("row %d: alt fill = %v, want %v: %s", index, got, want, first)
		}
	}
}

func TestATableWithoutZebraDrawsNoAltRows(t *testing.T) {
	if dump := Dump(Table(readOnlySpec())); strings.Contains(dump, "bg=alt") {
		t.Fatalf("an unset Zebra fills rows:\n%s", dump)
	}
}

func TestCellNodeReplacesTheTextOfACell(t *testing.T) {
	spec := readOnlySpec()
	spec.CellNode = func(row, col int) Node {
		if row == 1 && col == 1 {
			return Text("custom")
		}
		return nil
	}
	dump := Dump(Table(spec))
	if !strings.Contains(dump, `text="custom"`) || strings.Contains(dump, `text="2"`) {
		t.Fatalf("the node did not replace the cell text:\n%s", dump)
	}
	if !strings.Contains(dump, `text="1"`) {
		t.Fatalf("a cell the callback declines lost its text:\n%s", dump)
	}
}

func TestClickingARowReportsItWithoutAnEditableTable(t *testing.T) {
	var rows []int
	spec := readOnlySpec()
	spec.OnRowClick = func(row int) { rows = append(rows, row) }
	root := Col(Table(spec))
	ctx := testCtx(t)
	Layout(ctx, root, 400, 400)
	click(ctx, 100, cellY(2))
	Update(ctx, root)
	if len(rows) != 1 || rows[0] != 2 {
		t.Fatalf("clicked rows = %v, want [2]", rows)
	}
}

func leadSpec() (TableSpec, map[int]bool) {
	shown := map[int]bool{1: true}
	spec := readOnlySpec()
	spec.State = &TableState{}
	spec.RowLead = func(row int) Node {
		if shown[row] {
			return Text("L")
		}
		return nil
	}
	spec.RowDetail = func(row int) Node {
		if shown[row] {
			return Row(HOf(core.RowOf), Text("detail"))
		}
		return nil
	}
	return spec, shown
}

func TestRowLeadAndDetailKeepTheColumnsAligned(t *testing.T) {
	spec, _ := leadSpec()
	table := Table(spec)
	root := Col(table)
	ctx := testCtx(t)
	Layout(ctx, root, 400, 400)
	header := table.Children()[0]
	// Row 0 has no lead node, yet its cells start where the header's do.
	row := rowsOf(table)[0]
	headerCells := Dump(header)
	if !strings.Contains(headerCells, fmt.Sprintf("w=%d", tableActionW)) {
		t.Fatalf("the header has no spacer for the lead column:\n%s", headerCells)
	}
	rowCells := row.Children()
	if got, want := rowCells[1].Rect().X, tableActionW; got != want {
		t.Errorf("first cell of a row starts at x=%d, want %d (after the lead)", got, want)
	}
	if !strings.Contains(Dump(table), `text="detail"`) {
		t.Fatalf("the detail row is missing:\n%s", Dump(table))
	}
	if strings.Count(Dump(table), `text="detail"`) != 1 {
		t.Fatalf("detail shown for rows that have none:\n%s", Dump(table))
	}
}

func TestRowDetailSitsUnderItsRow(t *testing.T) {
	spec, _ := leadSpec()
	table := Table(spec)
	root := Col(table)
	ctx := testCtx(t)
	Layout(ctx, root, 400, 400)
	rows := rowsOf(table)
	if got, want := rows[2].Rect().Y-rows[1].Rect().Y, 2*core.RowH; got != want {
		t.Fatalf("the row after a detailed row is %dpx below it, want %d (row plus detail)", got, want)
	}
}

// With a lead column the divider handles must still sit on the header's
// cell boundaries: dragging the first divider changes the weights.
func TestDividerHandlesCountTheLeadColumn(t *testing.T) {
	spec, _ := leadSpec()
	table := Table(spec)
	root := Col(table)
	ctx := testCtx(t)
	Layout(ctx, root, 400, 400)
	boundary := tableActionW + (400-tableActionW)/2
	ctx.In = core.Input{X: boundary, Y: core.RowH / 2, Click: true, Held: true}
	Update(ctx, root)
	ctx.In = core.Input{X: boundary + 40, Y: core.RowH / 2, Held: true}
	Update(ctx, root)
	if len(spec.State.Weights) != 2 || spec.State.Weights[0] <= spec.State.Weights[1] {
		t.Fatalf("weights after dragging the divider right = %v, want the first column wider", spec.State.Weights)
	}
}

func TestKeyChipsSplitsTheKeyTextOnPlus(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"Ctrl+Shift+Tab", []string{"Ctrl", "Shift", "Tab"}},
		{"F5", []string{"F5"}},
		{"Ctrl++", []string{"Ctrl", "+"}},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			dump := Dump(KeyChips(tc.text))
			if got := strings.Count(dump, "control=button"); got != len(tc.want) {
				t.Fatalf("%d chips, want %d:\n%s", got, len(tc.want), dump)
			}
			for _, key := range tc.want {
				if !strings.Contains(dump, fmt.Sprintf("text=%q", key)) {
					t.Errorf("no chip for %q:\n%s", key, dump)
				}
			}
		})
	}
}
