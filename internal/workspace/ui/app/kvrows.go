package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// kvRows turns key/value rows into table rows with two or three cells.
func kvRows(rows []model.KV, withDesc, flagDuplicates bool) []widget.TableRow {
	out := make([]widget.TableRow, len(rows))
	var dup []bool
	if flagDuplicates {
		dup = model.DuplicateNames(rows)
	}
	for i, kv := range rows {
		cells := []string{kv.Name, kv.Value}
		if kv.Secret {
			cells[1] = strings.Repeat("•", min(utf8.RuneCountInString(kv.Value), 12))
		}
		if withDesc {
			cells = append(cells, kv.Description)
		}
		out[i] = widget.TableRow{Cells: cells, Checked: kv.Enabled, Muted: !kv.Enabled, Flagged: dup != nil && dup[i]}
	}
	return out
}
