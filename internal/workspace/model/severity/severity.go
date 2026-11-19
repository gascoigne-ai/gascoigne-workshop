// Package severity is how serious a message for the user is. It lives in
// the workspace model so the settings, history and trace views and the app
// share one word for it; the app maps it to the look of a toast.
package severity

import "fmt"

// Level is how serious a message is. The zero value is Info.
type Level uint8

// The levels, least serious first.
const (
	Info Level = iota
	Error
)

var names = [...]string{Info: "info", Error: "error"}

// String is the level's name.
func (level Level) String() string {
	if int(level) >= len(names) {
		return fmt.Sprintf("level(%d)", uint8(level))
	}
	return names[level]
}

// All lists every level, so a table keyed by Level can be checked for gaps.
func All() []Level { return []Level{Info, Error} }
