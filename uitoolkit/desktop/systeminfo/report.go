package systeminfo

import "strings"

// Row is one labelled value, such as "Graphics library" and "Metal".
type Row struct {
	Label, Value string
}

// Section is a titled group of rows.
type Section struct {
	Title string
	Rows  []Row
}

// Report is the whole report, in display order.
type Report struct {
	Sections []Section
}

// Text is the report as plain text, for the clipboard or a bug report: each
// section's title, then its rows as "label: value", with a blank line
// between sections.
func (r Report) Text() string {
	var b strings.Builder
	for i, s := range r.Sections {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(s.Title)
		b.WriteByte('\n')
		for _, row := range s.Rows {
			b.WriteString("  " + row.Label + ": " + row.Value + "\n")
		}
	}
	return b.String()
}

// Value is the value of the row with the label in any section, "" when there
// is none.
func (r Report) Value(label string) string {
	for _, s := range r.Sections {
		for _, row := range s.Rows {
			if row.Label == label {
				return row.Value
			}
		}
	}
	return ""
}
