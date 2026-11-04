package vml_test

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// A view is a function from state to a tree. Dump prints the tree, which
// is how tests check a view without a window.
func Example() {
	loggedIn := true
	view := vml.Col(vml.Pad(12), vml.Gap(8),
		vml.Text("Hello", vml.Muted()),
		vml.If(loggedIn, vml.Button("Log out", vml.W(80))),
		vml.For([]string{"a", "b"}, func(_ int, name string) vml.Node {
			return vml.Text(name, vml.Small())
		}),
	)
	fmt.Print(vml.Dump(view))
	// Output:
	// flow padL=12 padT=12 padR=12 padB=12 gap=8
	//   textB text="Hello" fg=muted
	//   flow w=80 bg=sunken control=button lighten=true
	//     textB text="Log out" grow=1 fg=chip-text center=true
	//   textB text="a" small=true
	//   textB text="b" small=true
}

// Options may stand anywhere in the list; a later one wins. Button uses
// this to let its caller override the defaults.
func ExampleButton() {
	b := vml.Button("Save", vml.Bg(core.Accent))
	fmt.Print(vml.Dump(b))
	// Output:
	// flow bg=accent control=button lighten=true
	//   textB text="Save" grow=1 fg=chip-text center=true
}

// Widgets keep their state in plain structs the application owns; a
// field edits its text in place.
func ExampleTextField() {
	f := vml.NewTextField("hello")
	f.SelectAll()
	f.Insert("world")
	fmt.Println(f.Text)
	// Output: world
}
