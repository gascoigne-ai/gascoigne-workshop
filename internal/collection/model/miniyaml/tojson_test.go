package miniyaml

import (
	"errors"
	"testing"
)

func TestToJSONWritesTheDocumentAsJSON(t *testing.T) {
	got, err := ToJSON([]byte("a: 1\nb:\n  - x\n  - true\n"))
	if err != nil || string(got) != `{"a":1,"b":["x",true]}` {
		t.Fatalf("ToJSON = %s, %v", got, err)
	}
}

func TestToJSONReportsAProblemWithItsLine(t *testing.T) {
	_, err := ToJSON([]byte("a: 1\n\tb: 2\n"))
	var yamlErr *Error
	if !errors.As(err, &yamlErr) || yamlErr.Line != 2 {
		t.Fatalf("err = %v, want an *Error at line 2", err)
	}
}
