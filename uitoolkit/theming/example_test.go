package theming_test

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
)

// Preferences in, theme out. Unknown ids fall back to the first built-in
// theme; high contrast and color overrides are checked for readability.
func Example() {
	catalog := theming.NewCatalog(theming.Builtins())
	th := catalog.Theme(theming.Prefs{Theme: "light", TextSize: "large", Corners: "round", Contrast: "high"})
	fmt.Println(th.Name, th.RadiusScale)
	fmt.Println(catalog.ThemeID("no-such-theme"))
	for _, c := range catalog.All() {
		fmt.Println(c.ID, "-", c.Label)
	}
	// Output:
	// light 1.5
	// dark
	// dark - Dark
	// light - Light
}

// A theme is data: colors by token name, resolved to RGBA.
func ExampleTheme_Color() {
	th := theming.Dark()
	c := th.Color(core.Accent)
	fmt.Println(theming.FormatHex(c) != "")
	// Output: true
}
