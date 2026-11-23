package kvtable

import (
	"strings"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// kvRows turns key/value rows into table rows of a name, a value and a
// description.
func kvRows(rows []request.KV, flagDuplicates bool) []widget.TableRow {
	out := make([]widget.TableRow, len(rows))
	var dup []bool
	if flagDuplicates {
		dup = request.DuplicateNames(rows)
	}
	for i, kv := range rows {
		cells := []string{kv.Name, kv.Value, kv.Description}
		if kv.Secret {
			cells[1] = strings.Repeat("•", min(utf8.RuneCountInString(kv.Value), 12))
		}
		out[i] = widget.TableRow{Cells: cells, Checked: kv.Enabled, Muted: !kv.Enabled, Flagged: dup != nil && dup[i]}
	}
	return out
}
