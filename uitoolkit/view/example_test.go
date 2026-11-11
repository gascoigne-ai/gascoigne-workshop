package view_test

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// A view is a function from state to a tree. Dump prints the tree, which
// is how tests check a view without a window.
func Example() {
	loggedIn := true
	tree := view.Col(view.Pad(12), view.Gap(8),
		view.Text("Hello", view.Muted()),
		view.If(loggedIn, view.Text("Logged in", view.Bold())),
		view.For([]string{"a", "b"}, func(_ int, name string) view.Node {
			return view.Text(name, view.Small())
		}),
	)
	fmt.Print(view.Dump(tree))
	// Output:
	// flow padL=12 padT=12 padR=12 padB=12 gap=8
	//   textB text="Hello" fg=muted
	//   textB text="Logged in" bold=true
	//   textB text="a" small=true
	//   textB text="b" small=true
}

// Widgets keep their state in plain structs the application owns; a
// field edits its text in place.
func ExampleTextField() {
	f := view.NewTextField("hello")
	f.SelectAll()
	f.Insert("world")
	fmt.Println(f.Text)
	// Output: world
}
