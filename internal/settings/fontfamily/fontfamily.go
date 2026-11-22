// Package fontfamily is a typeface by name and the rules to find one in a
// list; reading the files is the font adapter's job.
package fontfamily

import "strings"

// Family is a typeface by name with the files of its regular and bold
// styles. Bold is empty when the family has none.
type Family struct {
	Name          string
	Regular, Bold string
}

// Find is the family called name, ignoring case.
func Find(families []Family, name string) (Family, bool) {
	for _, family := range families {
		if strings.EqualFold(family.Name, strings.TrimSpace(name)) {
			return family, true
		}
	}
	return Family{}, false
}

// Matching lists up to limit families whose name contains text, ignoring
// case; empty text matches all.
func Matching(families []Family, text string, limit int) []Family {
	needle := strings.ToLower(strings.TrimSpace(text))
	var out []Family
	for _, family := range families {
		if len(out) == limit {
			break
		}
		if strings.Contains(strings.ToLower(family.Name), needle) {
			out = append(out, family)
		}
	}
	return out
}
