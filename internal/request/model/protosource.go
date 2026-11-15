package model

import "fmt"

// ProtoError is a problem in a .proto file, at the line and column where
// it was found (both from 1), so the user can go to it.
type ProtoError struct {
	Path      string
	Line, Col int
	Message   string
}

func (e ProtoError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Col, e.Message)
}
