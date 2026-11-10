// Package viewer holds the rules of how a response is looked at: the
// response view choices, and the counter and stepping of the find bar.
package viewer

import "fmt"

// Views are the values of the response view setting.
var Views = []struct {
	ID, Label    string
	Raw, Compact bool
}{
	{"compact", "Compact", false, true},
	{"pretty", "Pretty", false, false},
	{"raw", "Raw", true, false},
}

// ModeOf is the mode of a response view setting; anything unknown is
// compact, the default.
func ModeOf(id string) Mode {
	raw, compact := Flags(id)
	switch {
	case raw:
		return Raw
	case compact:
		return Compact
	}
	return Pretty
}

// Flags are the raw and compact flags of a response view setting; anything
// unknown is compact, the default.
func Flags(id string) (raw, compact bool) {
	for _, view := range Views {
		if view.ID == id {
			return view.Raw, view.Compact
		}
	}
	return false, true
}

// Count is the counter shown in the find bar: "3/12", "No matches", "3
// matches" when none is active, or nothing for an empty query.
func Count(query string, matches, index int) string {
	switch {
	case query == "":
		return ""
	case matches == 0:
		return "No matches"
	case index < 0:
		return fmt.Sprintf("%d matches", matches)
	}
	return fmt.Sprintf("%d/%d", index+1, matches)
}

// Step is the match after moving dir (1 or -1) from index among matches,
// wrapping at the ends. With no match active, moving back goes to the last.
// With no matches it stays at index.
func Step(index, dir, matches int) int {
	if matches == 0 {
		return index
	}
	if index < 0 && dir < 0 {
		return matches - 1
	}
	return (index + dir + matches) % matches
}
