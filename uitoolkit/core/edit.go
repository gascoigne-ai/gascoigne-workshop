package core

// Edit is a cursor or deletion key applied to a TextField.
type Edit int

const (
	EditBackspace Edit = iota
	EditDelete
	EditLeft
	EditRight
	EditHome
	EditEnd
	EditWordLeft
	EditWordRight
	EditDeleteWordLeft
	EditDeleteWordRight
	EditUp
	EditDown
	EditPageUp
	EditPageDown
	EditDocStart
	EditDocEnd
	EditNewline
	EditIndent
	EditDedent
	// EditDeleteToLineStart deletes from the line start to the cursor
	// (Cmd+Backspace on macOS); at the line start it takes the line break.
	EditDeleteToLineStart
	// EditDeleteToLineEnd deletes from the cursor to the line end
	// (Cmd+Delete on macOS); at the line end it takes the line break.
	EditDeleteToLineEnd
)
