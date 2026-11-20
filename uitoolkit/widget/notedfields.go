package widget

// NotedFields is what a settings-like form keeps per field: the text field a
// user types into and the note that says why the last typed value was
// refused. The zero value is ready.
type NotedFields[K comparable] struct {
	inputs map[K]*TextField
	notes  map[K]string
}

// Input is the text field of key. initial gives its first text and runs only
// when the field is created; the pointer stays the same until DropInput or
// Forget.
func (n *NotedFields[K]) Input(key K, initial func() string) *TextField {
	if input, ok := n.inputs[key]; ok {
		return input
	}
	if n.inputs == nil {
		n.inputs = map[K]*TextField{}
	}
	input := new(TextField)
	*input = NewTextField(initial())
	n.inputs[key] = input
	return input
}

// Note is why the value typed for key was refused, "" when it was not.
func (n *NotedFields[K]) Note(key K) string { return n.notes[key] }

// SetNote sets the refusal note of key; "" clears it.
func (n *NotedFields[K]) SetNote(key K, text string) {
	if text == "" {
		delete(n.notes, key)
		return
	}
	if n.notes == nil {
		n.notes = map[K]string{}
	}
	n.notes[key] = text
}

// DropInput forgets the text field of key, so it is rebuilt from its initial
// text; the note stays.
func (n *NotedFields[K]) DropInput(key K) { delete(n.inputs, key) }

// Forget drops every field and note.
func (n *NotedFields[K]) Forget() { n.inputs, n.notes = nil, nil }
