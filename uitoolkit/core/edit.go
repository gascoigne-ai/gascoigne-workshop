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
)
