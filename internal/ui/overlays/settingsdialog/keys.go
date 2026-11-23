package settingsdialog

import w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// Recording reports whether the Shortcuts page waits for the key of an
// action. While it does, every key belongs to the recording.
func (c *Component) Recording() bool { return c.Shortcuts.Recording() != "" }

// shortcutsPageShown reports whether the Shortcuts page is what the dialog
// draws: not another page, the raw text or a search. Only then do its rows
// take the arrow keys and Enter.
func (c *Component) shortcutsPageShown() bool {
	return c.open && c.Shell.Cat == "shortcuts" && !c.Shell.Raw && !c.Shell.Searching()
}

// Keys takes the keys of the frame that belong to the dialog and reports
// whether it did: the recorded key while an action is recording, else the
// arrow keys and Enter of the Shortcuts page while it is shown. A false
// answer leaves the keys to the keymap.
func (c *Component) Keys(in w.Input) bool {
	if c.Recording() {
		c.Shortcuts.RecordKey(c, in)
		return true
	}
	if c.shortcutsPageShown() {
		c.Shortcuts.Navigate(in, c)
		return true
	}
	return false
}
