package viewer

// Mode is how the response body is shown.
type Mode int

// The response modes: pretty-printed, compact JSON, as received.
const (
	Pretty Mode = iota
	Compact
	Raw
)

// Model is what the response pane of one tab remembers: the mode, whether
// the find bar is showing and the active match (-1 for none). The text
// searched and the lines shown belong to the widgets.
type Model struct {
	Mode     Mode
	FindOpen bool
	Match    int
}

// New is the model of a fresh tab for a response view setting.
func New(setting string) Model {
	return Model{Mode: ModeOf(setting), Match: -1}
}

// Msg is something that happens to the response pane.
type Msg interface{ viewerMsg() }

// ModeChosen picks how the body is shown.
type ModeChosen struct{ Mode Mode }

// FindOpened shows the find bar.
type FindOpened struct{}

// FindClosed hides the find bar and drops the active match.
type FindClosed struct{}

// Searched reports that the view was searched again (the text changed or
// the response was redrawn) and found Matches; the first one is active.
type Searched struct{ Matches int }

// Stepped moves the active match by Dir (1 or -1) among Matches.
type Stepped struct{ Dir, Matches int }

func (ModeChosen) viewerMsg() {}
func (FindOpened) viewerMsg() {}
func (FindClosed) viewerMsg() {}
func (Searched) viewerMsg()   {}
func (Stepped) viewerMsg()    {}

// Update is the model after msg.
func Update(model Model, msg Msg) Model {
	switch msg := msg.(type) {
	case ModeChosen:
		model.Mode = msg.Mode
	case FindOpened:
		model.FindOpen = true
	case FindClosed:
		model.FindOpen, model.Match = false, -1
	case Searched:
		model.Match = -1
		if msg.Matches > 0 {
			model.Match = 0
		}
	case Stepped:
		model.Match = Step(model.Match, msg.Dir, msg.Matches)
	}
	return model
}

// Raw reports whether the body is shown as received.
func (model Model) Raw() bool { return model.Mode == Raw }

// Compact reports whether JSON is shown compactly.
func (model Model) Compact() bool { return model.Mode == Compact }
