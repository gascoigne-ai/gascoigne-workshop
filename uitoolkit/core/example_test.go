package core_test

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Widgets keep their state in plain structs the application owns; a
// field edits its text in place.
func ExampleTextField() {
	f := core.NewTextField("hello")
	f.SelectAll()
	f.Insert("world")
	fmt.Println(f.Text)
	// Output: world
}

// A theme is data: colors by token name, resolved to RGBA.
func ExampleTheme_Color() {
	th := core.Dark()
	c := th.Color(core.Accent)
	fmt.Println(core.FormatHex(c) != "")
	// Output: true
}
