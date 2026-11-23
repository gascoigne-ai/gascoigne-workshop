package statusbar

// Deps is what the status bar asks of the application, each a narrow
// interface the bar declares itself so it imports no controller.
type Deps struct {
	Status  StatusLine
	Log     Log
	Welcome Welcome
	Version string
}

// StatusLine is the text of the line, read every frame.
type StatusLine interface{ Status() string }

// Log opens the log of calls.
type Log interface{ Open() }

// Welcome opens the release notes.
type Welcome interface{ OpenWelcome() }
