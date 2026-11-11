package examples_test

import (
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// A view is a function from state to a tree. Dump prints the tree, which
// is how tests check a view without a window.
func Example_view() {
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
func Example_textField() {
	f := widget.NewTextField("hello")
	f.SelectAll()
	f.Insert("world")
	fmt.Println(f.Text)
	// Output: world
}

// Options may stand anywhere in the list; a later one wins. Button uses
// this to let its caller override the defaults.
func Example_button() {
	b := widget.Button("Save", view.Bg(core.Accent))
	fmt.Print(view.Dump(b))
	// Output:
	// flow padL=10 padR=10 bg=accent control=button lighten=true
	//   textB text="Save" grow=1 fg=chip-text ellipsis=true center=true
}

// A template written inline in the method that feeds it.
func Example_markupView() {
	status := "Ready"
	node := markup.View(`
		<row h="28" pad="12 0" bg="frame">
		  <text small muted>{{status}}</text>
		  <text small if="saved">saved</text>
		</row>`,
		markup.Scope{"status": status, "saved": true})
	fmt.Print(view.Dump(node))
	// Output:
	// flow h=28 padL=12 padR=12 bg=frame
	//   textB text="Ready" fg=muted small=true
	//   textB text="saved" small=true
}

// A repeated element reads each item of a list.
func Example_markupParse() {
	t, err := markup.Parse("tabs.vml", []byte(`
		<row gap="4">
		  <button for="tab in tabs" on-click="pick">{{tab.name}}</button>
		</row>`))
	if err != nil {
		panic(err)
	}
	node, err := t.Build(markup.Scope{
		"tabs": []map[string]any{{"name": "Params"}, {"name": "Body"}},
		"pick": func() {},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(strings.Count(view.Dump(node), "text="), "labels")
	// Output: 2 labels
}

// Mistakes name the file and line, suggest the nearest name and list
// the valid ones.
func Example_markupParseErrors() {
	_, err := markup.Parse("bar.vml", []byte("<row>\n  <txt/>\n</row>"))
	fmt.Println(strings.SplitN(err.Error(), ";", 2)[0])
	// Output: bar.vml:2: unknown element <txt> (did you mean "text"?)
}

// Preferences in, theme out. Unknown ids fall back to the first built-in
// theme; high contrast and color overrides are checked for readability.
func Example_theming() {
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
func Example_themeColor() {
	th := theming.Dark()
	c := th.Color(core.Accent)
	fmt.Println(core.FormatHex(c) != "")
	// Output: true
}
