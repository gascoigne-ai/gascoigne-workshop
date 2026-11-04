package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// kvRows turns key/value rows into table rows with two or three cells.
func kvRows(rows []model.KV, withDesc, flagDuplicates bool) []vml.TableRow {
	out := make([]vml.TableRow, len(rows))
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
		out[i] = vml.TableRow{Cells: cells, Checked: kv.Enabled, Muted: !kv.Enabled, Flagged: dup != nil && dup[i]}
	}
	return out
}
