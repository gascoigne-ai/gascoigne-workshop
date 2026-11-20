package widget

import "testing"

func TestNotedFieldsInputIsCreatedOnceAndKeepsItsIdentity(t *testing.T) {
	var fields NotedFields[string]
	created := 0
	initial := func() string { created++; return "first" }
	one := fields.Input("a", initial)
	two := fields.Input("a", initial)
	if one == nil || one != two || created != 1 || one.Text != "first" {
		t.Fatalf("one %p two %p created %d", one, two, created)
	}
	if other := fields.Input("b", func() string { return "x" }); other == one {
		t.Fatal("two keys share one field")
	}
}

func TestNotedFieldsNotes(t *testing.T) {
	var fields NotedFields[string]
	if fields.Note("a") != "" {
		t.Fatalf("note of an untouched key %q", fields.Note("a"))
	}
	fields.SetNote("a", "bad")
	if fields.Note("a") != "bad" {
		t.Fatalf("note %q", fields.Note("a"))
	}
	fields.SetNote("a", "")
	if fields.Note("a") != "" {
		t.Fatalf("note after clearing %q", fields.Note("a"))
	}
}

func TestNotedFieldsDropInputKeepsNotesAndForgetClearsBoth(t *testing.T) {
	var fields NotedFields[int]
	first := fields.Input(1, func() string { return "x" })
	fields.SetNote(1, "bad")
	fields.DropInput(1)
	if fields.Note(1) != "bad" {
		t.Fatalf("DropInput lost the note: %q", fields.Note(1))
	}
	if again := fields.Input(1, func() string { return "y" }); again == first || again.Text != "y" {
		t.Fatal("DropInput kept the field")
	}
	fields.Forget()
	if fields.Note(1) != "" {
		t.Fatal("Forget kept the note")
	}
	created := false
	fields.Input(1, func() string { created = true; return "" })
	if !created {
		t.Fatal("Forget kept the field")
	}
}
