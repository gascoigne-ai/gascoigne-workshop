package model

import "testing"

func TestProtoErrorNamesFileLineAndColumn(t *testing.T) {
	err := ProtoError{Path: "/protos/health.proto", Line: 34, Col: 1, Message: "syntax error: unexpected EOF"}
	if got, want := err.Error(), "/protos/health.proto:34:1: syntax error: unexpected EOF"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
