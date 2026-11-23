// Package keymap resolves which key runs which command, and runs it. The
// components offer commands (an id, a name, a default chord, what to do); the
// settings hold the user's overrides as text; Resolve layers the one over the
// other, so a key press, the Shortcuts page and a tooltip all read one answer.
//
// The package knows no toolkit: a Chord is plain data, parsing the settings
// text is passed in, and whether a key is also typed is told by the caller in
// the Press. That keeps it in the controller zone (settings root and base
// only) and testable without a window.
//
// The keymap is derived from the settings and the commands, never stored: the
// caller resolves it where it needs it (once per key frame, once per draw), so
// a changed setting needs no invalidation. Rejected: a keymap kept on the App
// and rebuilt on every settings change, which would be a second copy of a fact
// the settings already hold; revisit when resolving shows up in a profile.
package keymap

// ID names a command in the settings file.
type ID string

// Chord is a key with modifiers. Key is the toolkit's key code, and 0 is a real
// key, so an unbound command is the absence of a chord, never a zero Chord.
type Chord struct {
	Key              int
	Ctrl, Shift, Alt bool
}

// Command is something a key can do. Components offer them; the App offers
// the ones that span components.
type Command struct {
	ID      ID
	Name    string // as users read it
	Default Chord  // the key until the user rebinds it
	Run     func()
}

// Parse reads the text of a chord from the settings. "" is a valid, unbound
// answer (bound false); text that is not a chord is an error.
type Parse func(text string) (chord Chord, bound bool, err error)

// Press is a key that went down: its chord and whether the key is also one the
// user types with, which the caller knows from the toolkit.
type Press struct {
	Chord     Chord
	TypesText bool
}

// Keymap is the resolved binding of every command, in display order.
type Keymap struct {
	commands []Command
	chords   map[ID]Chord // bound commands only
	user     map[ID]bool  // commands whose binding is the user's
	byChord  map[Chord][]int
}

// Resolve layers the user's overrides over the commands' defaults. A valid
// override wins, "" unbinds, text that does not parse is ignored and an id no
// command has is skipped. Two commands may share a chord when a hand-edited
// settings file says so; both run on it.
func Resolve(overrides map[string]string, commands []Command, parse Parse) Keymap {
	keys := Keymap{commands: commands, chords: map[ID]Chord{}, user: map[ID]bool{}, byChord: map[Chord][]int{}}
	for index, command := range commands {
		chord, bound := command.Default, true
		if text, ok := overrides[string(command.ID)]; ok {
			if parsed, parsedBound, err := parse(text); err == nil {
				chord, bound = parsed, parsedBound
				keys.user[command.ID] = true
			}
		}
		if bound {
			keys.chords[command.ID] = chord
			keys.byChord[chord] = append(keys.byChord[chord], index)
		}
	}
	return keys
}

// Commands are the commands in display order.
func (k Keymap) Commands() []Command { return k.commands }

// Chord is the chord of command id, false when it is unbound or unknown.
func (k Keymap) Chord(id ID) (Chord, bool) {
	chord, ok := k.chords[id]
	return chord, ok
}

// Overridden reports whether the binding of command id is the user's.
func (k Keymap) Overridden(id ID) bool { return k.user[id] }

// Conflict is the first other command that has chord, in display order.
func (k Keymap) Conflict(id ID, chord Chord) (Command, bool) {
	for _, index := range k.byChord[chord] {
		if k.commands[index].ID != id {
			return k.commands[index], true
		}
	}
	return Command{}, false
}

// Run runs every command on the pressed chord, in display order, and reports
// whether any ran. A key the user also types runs nothing while a text field
// has the keyboard.
func (k Keymap) Run(press Press, textFocused bool) bool {
	if press.TypesText && textFocused {
		return false
	}
	indexes := k.byChord[press.Chord]
	for _, index := range indexes {
		k.commands[index].Run()
	}
	return len(indexes) > 0
}
