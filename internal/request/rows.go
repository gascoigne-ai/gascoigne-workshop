package request

import (
	"strings"
	"unicode"
)

// InsertRow returns rows with kv inserted at index at (clamped). The
// input slice is never changed.
func InsertRow(rows []KV, at int, kv KV) []KV {
	at = min(max(at, 0), len(rows))
	out := make([]KV, 0, len(rows)+1)
	out = append(out, rows[:at]...)
	out = append(out, kv)
	return append(out, rows[at:]...)
}

// DuplicateRow returns rows with a copy of row i placed right after it.
func DuplicateRow(rows []KV, index int) []KV {
	if index < 0 || index >= len(rows) {
		return append([]KV(nil), rows...)
	}
	return InsertRow(rows, index+1, rows[index])
}

// DeleteRow returns rows without row i; an index out of range changes nothing.
func DeleteRow(rows []KV, index int) []KV {
	out := append([]KV(nil), rows...)
	if index < 0 || index >= len(out) {
		return out
	}
	return append(out[:index], out[index+1:]...)
}

// MoveRow moves row i by delta places (negative is up), stopping at the
// ends. It returns the new rows and the row's new index.
func MoveRow(rows []KV, index, delta int) ([]KV, int) {
	out := append([]KV(nil), rows...)
	if index < 0 || index >= len(out) {
		return out, index
	}
	to := min(max(index+delta, 0), len(out)-1)
	row := out[index]
	out = append(out[:index], out[index+1:]...)
	return InsertRow(out, to, row), to
}

// SetAll returns rows with every Enabled set to on.
func SetAll(rows []KV, on bool) []KV {
	out := append([]KV(nil), rows...)
	for i := range out {
		out[i].Enabled = on
	}
	return out
}

// AllEnabled reports whether there is at least one row and all are enabled.
func AllEnabled(rows []KV) bool {
	for _, r := range rows {
		if !r.Enabled {
			return false
		}
	}
	return len(rows) > 0
}

// DuplicateNames returns, per row, whether another enabled row has the
// same name (case-insensitive). Disabled and unnamed rows are never flagged.
func DuplicateNames(rows []KV) []bool {
	count := map[string]int{}
	for _, r := range rows {
		if r.Enabled && r.Name != "" {
			count[strings.ToLower(r.Name)]++
		}
	}
	flags := make([]bool, len(rows))
	for i, r := range rows {
		flags[i] = r.Enabled && r.Name != "" && count[strings.ToLower(r.Name)] > 1
	}
	return flags
}

// FormatBulk renders rows as text for bulk editing: "name: value" per
// row, with disabled rows prefixed "# ".
func FormatBulk(rows []KV) string {
	lines := make([]string, len(rows))
	for i, r := range rows {
		line := r.Name + ": " + r.Value
		if !r.Enabled {
			line = "# " + line
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// ParseBulk turns bulk text back into rows. Blank lines are skipped,
// "# " marks a disabled row, the name ends at the first colon and one
// space after it is dropped. Descriptions carry over from the rows of
// old with the same name, each used once.
func ParseBulk(text string, old []KV) []KV {
	used := make([]bool, len(old))
	var rows []KV
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		enabled := true
		if trimmed := strings.TrimLeftFunc(line, unicode.IsSpace); strings.HasPrefix(trimmed, "#") {
			enabled = false
			line = strings.TrimPrefix(strings.TrimPrefix(trimmed, "#"), " ")
		}
		name, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		row := KV{Name: strings.TrimSpace(name), Value: value, Enabled: enabled}
		for i, o := range old {
			if !used[i] && o.Name == row.Name {
				used[i], row.Description = true, o.Description
				break
			}
		}
		rows = append(rows, row)
	}
	return rows
}
