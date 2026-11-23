// Package logic is what the key-value tables decide without a screen: which
// table a cell belongs to, where Tab goes, what a committed cell does to the
// rows and which row the keyboard acts on. The component in the parent
// directory draws and holds the state; these are plain functions over it.
package logic

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Kind is which table of a tab: the zero value is none.
type Kind int

const (
	NoTable Kind = iota
	Params
	Headers
	Vars
	Form
	ResponseHeaders
	KindCount
)

// kinds is what each Kind is called: the label names its pane and keys its
// undo history, the key files its column widths (the request's and the
// response's Headers tables share a label, not a key).
var kinds = [KindCount]struct{ label, key string }{
	Params:          {"Params", "Params"},
	Headers:         {"Headers", "Headers"},
	Vars:            {"Vars", "Vars"},
	Form:            {"Body", "Body"},
	ResponseHeaders: {"Headers", "response Headers"},
}

// Label is the key of the table's undo history and the name of its pane.
func (kind Kind) Label() string { return kinds[kind].label }

// Key is what the remembered column widths of the table are filed under.
func (kind Kind) Key() string { return kinds[kind].key }

// KindOfKey is the table whose widths are filed under key.
func KindOfKey(key string) (Kind, bool) {
	for kind := Params; kind < KindCount; kind++ {
		if kind.Key() == key {
			return kind, true
		}
	}
	return NoTable, false
}

// Table is one table of one tab.
type Table struct {
	Tab  *tabs.Tab
	Kind Kind
}

// Cell is one cell of a table; Row may be the placeholder row, one past the
// last.
type Cell struct {
	Table
	Row, Col int
}

// EditColumns is the number of editable columns of a table: name, value,
// description.
const EditColumns = 3

// NextCell is the cell Tab moves to from cell (the previous one with back),
// wrapping across rows and stopping in the placeholder row, whose index is
// rows.
func NextCell(cell Cell, back bool, rows int) Cell {
	if back {
		cell.Col--
		if cell.Col < 0 {
			cell.Col, cell.Row = EditColumns-1, cell.Row-1
		}
	} else {
		cell.Col++
		if cell.Col >= EditColumns {
			cell.Col, cell.Row = 0, cell.Row+1
		}
	}
	if cell.Row < 0 {
		cell.Row, cell.Col = 0, 0
	}
	if cell.Row > rows {
		cell.Row, cell.Col = rows, EditColumns-1
	}
	return cell
}

// kvField points at the name, value or description of a row.
func kvField(kv *request.KV, col int) *string {
	return [EditColumns]*string{&kv.Name, &kv.Value, &kv.Description}[col]
}

// CellText is the text of a cell of rows, "" for the placeholder row.
func CellText(rows []request.KV, row, col int) string {
	if row >= len(rows) {
		return ""
	}
	return *kvField(&rows[row], col)
}

// SetCell writes text into a cell of rows. Text typed into the placeholder
// row (or a row the rows lost since the edit began) adds an enabled row; an
// empty placeholder cell adds nothing.
func SetCell(rows []request.KV, row, col int, text string) []request.KV {
	if row >= len(rows) {
		if text == "" {
			return rows
		}
		rows = append(rows, request.KV{Enabled: true})
		row = len(rows) - 1
	}
	*kvField(&rows[row], col) = text
	return rows
}

// Settle drops the edit and the selection of table that point past its rows
// after they changed. The placeholder row, index rows, may still be edited
// but not selected.
func Settle(edit, selected *Cell, table Table, rows int) (*Cell, *Cell) {
	if edit != nil && edit.Table == table && edit.Row > rows {
		edit = nil
	}
	if selected != nil && selected.Table == table && selected.Row >= rows {
		selected = nil
	}
	return edit, selected
}

// Target is the row the keyboard operations act on: the edited row, or the
// selected one while its table is the visible one. An edit past the rows
// gives none, and the selection is not a fallback for it.
func Target(edit, selected *Cell, visible Table, shown bool, rows func(Table) int) (cell Cell, editing, ok bool) {
	if edit != nil {
		if edit.Row < rows(edit.Table) {
			return *edit, true, true
		}
		return Cell{}, false, false
	}
	if selected != nil && shown && selected.Table == visible && selected.Row < rows(visible) {
		return Cell{Table: visible, Row: selected.Row}, false, true
	}
	return Cell{}, false, false
}

// AfterDelete is the row to select once a row was deleted and rows are left,
// none when the table is empty.
func AfterDelete(row, rows int) (int, bool) {
	if rows == 0 {
		return 0, false
	}
	return min(row, rows-1), true
}

// SecretLabel is the menu entry that flips a variable's secret mark.
func SecretLabel(secret bool) string {
	if secret {
		return "Not a secret"
	}
	return "Mark as secret"
}

// BulkLabel is the link that switches between the table and its text.
func BulkLabel(inBulk bool) string {
	if inBulk {
		return "Table"
	}
	return "Bulk edit"
}
