package settings

// LayoutChoice is an entry of the layout chooser in the settings menu.
type LayoutChoice struct{ ID, Label string }

// LayoutChoices are the pane layouts a Settings.Layout may name, the
// default first.
var LayoutChoices = []LayoutChoice{
	{"stacked", "Stacked"},
	{"side", "Side by side"},
}

// The interface scale slider's range and step, in percent.
const (
	ScaleSliderMin  = 50
	ScaleSliderMax  = 200
	ScaleSliderStep = 5
)
