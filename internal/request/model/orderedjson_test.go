package model

import "testing"

// Pins the two ordered-object writers that share writeObject: arguments are
// written without HTML escapes, skeleton members keep the escapes of
// encoding/json.
func TestOrderedObjectWritersKeepTheirEscaping(t *testing.T) {
	fields := []ToolField{{Name: "b", Kind: FieldString, Required: true}, {Name: "a", Kind: FieldString, Required: true}}
	got, err := Arguments(fields, map[string]string{"a": "<x>", "b": "&"})
	if err != nil || got != `{"b":"&","a":"<x>"}` {
		t.Errorf("Arguments = %s, %v", got, err)
	}
	schema, err := DecodeJSONSchema([]byte(`{"type":"object","properties":{"q":{"type":"string","default":"<x>"},"z":{"type":"boolean","default":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := Skeleton(schema); got != `{"q":"`+"\\u003cx\\u003e"+`","z":true}` {
		t.Errorf("Skeleton = %q", got)
	}
}
