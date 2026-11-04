package core_test

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// A theme is data: colors by token name, resolved to RGBA.
func ExampleTheme_Color() {
	th := core.Dark()
	c := th.Color(core.Accent)
	fmt.Println(core.FormatHex(c) != "")
	// Output: true
}
