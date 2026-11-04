package widget_test

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widgets keep their state in plain structs the application owns; a
// field edits its text in place.
func ExampleTextField() {
	f := widget.NewTextField("hello")
	f.SelectAll()
	f.Insert("world")
	fmt.Println(f.Text)
	// Output: world
}

// A theme is data: colors by token name, resolved to RGBA.
func ExampleTheme_Color() {
	th := widget.Dark()
	c := th.Color(widget.Accent)
	fmt.Println(widget.FormatHex(c) != "")
	// Output: true
}
