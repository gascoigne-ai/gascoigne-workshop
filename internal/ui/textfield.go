package ui

// Edit is a cursor or deletion key applied to a TextField.
type Edit int

const (
	EditBackspace Edit = iota
	EditDelete
	EditLeft
	EditRight
	EditHome
	EditEnd
)

// TextField is a single-line editable string. Cursor counts runes.
type TextField struct {
	Text    string
	Cursor  int
	Focused bool
}

// NewTextField returns a field holding text with the cursor at its end.
func NewTextField(text string) TextField {
	return TextField{Text: text, Cursor: len([]rune(text))}
}

// Apply inserts typed runes at the cursor, then runs edits in order.
// It reports whether Text changed.
func (f *TextField) Apply(typed []rune, edits []Edit) (changed bool) {
	runes := []rune(f.Text)
	f.Cursor = min(max(f.Cursor, 0), len(runes))
	for _, r := range typed {
		if r < ' ' || r == 0x7f {
			continue
		}
		runes = append(runes[:f.Cursor], append([]rune{r}, runes[f.Cursor:]...)...)
		f.Cursor++
		changed = true
	}
	for _, e := range edits {
		switch e {
		case EditBackspace:
			if f.Cursor > 0 {
				runes = append(runes[:f.Cursor-1], runes[f.Cursor:]...)
				f.Cursor--
				changed = true
			}
		case EditDelete:
			if f.Cursor < len(runes) {
				runes = append(runes[:f.Cursor], runes[f.Cursor+1:]...)
				changed = true
			}
		case EditLeft:
			f.Cursor = max(f.Cursor-1, 0)
		case EditRight:
			f.Cursor = min(f.Cursor+1, len(runes))
		case EditHome:
			f.Cursor = 0
		case EditEnd:
			f.Cursor = len(runes)
		}
	}
	f.Text = string(runes)
	return changed
}
