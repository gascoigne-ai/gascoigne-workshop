package form

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"

// Place is where a row puts its control: beside the title block, or
// under it when the control needs the width of the row.
type Place int

const (
	PlaceByKind Place = iota // from the Kind of the setting (see PlaceOf)
	PlaceRight
	PlaceBelow
)

// customPlace is the placement of a setting whose control is in
// settingControls and is not chosen by Kind alone: a stepper is short, the
// rest of them need the width. A key that is not here goes by Kind.
var customPlace = map[string]Place{
	"smallFontSize": PlaceRight,
	"theme":         PlaceBelow, // one chip per installed theme
	"uiScale":       PlaceBelow, // a slider wants a long track
}

// shortChoices is how many values an enum may have for its chips to sit
// beside the title; more of them go under it, as a long row of chips would
// squeeze the description.
const shortChoices = 3

// PlaceOf is where the control of field goes. Why by Kind and not a
// per-key table: a new boolean or short enum lands right without a row here
// (revisit when a kind needs both places).
func PlaceOf(field settings.Field) Place {
	if place, ok := customPlace[field.Key]; ok {
		return place
	}
	switch {
	case field.Kind == settings.Bool:
		return PlaceRight
	case field.Kind == settings.Enum && !field.Open && len(field.Allowed) <= shortChoices:
		return PlaceRight
	}
	return PlaceBelow
}
