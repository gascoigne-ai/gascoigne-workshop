package settings

import (
	"fmt"
	"strings"
)

// ClampedError reports that a settings file held values outside the
// schema's ranges and that they were brought to the nearest allowed value.
// The settings returned with it are valid and meant to be used.
type ClampedError struct{ Notes []string }

func (err *ClampedError) Error() string {
	return "settings.json: " + strings.Join(err.Notes, "; ")
}

// Clamp brings the whole-number settings inside the ranges of the schema.
// 0 means "not set" for both and stays. notes tells what changed, one
// entry per key.
func Clamp(prefs Settings) (clamped Settings, notes []string) {
	prefs.TimeoutSec, notes = clampInt(prefs.TimeoutSec, "timeoutSec", notes)
	prefs.UIScalePct, notes = clampInt(prefs.UIScalePct, "uiScale", notes)
	return prefs, notes
}

func clampInt(value int, key string, notes []string) (int, []string) {
	field, ok := FieldFor(key)
	if !ok || value == 0 || (value >= field.Min && value <= field.Max) {
		return value, notes
	}
	bounded := max(field.Min, min(value, field.Max))
	return bounded, append(notes, fmt.Sprintf("%s %d is outside %d to %d, using %d", key, value, field.Min, field.Max, bounded))
}
