package widget_test

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Options may stand anywhere in the list; a later one wins. Button uses
// this to let its caller override the defaults.
func ExampleButton() {
	b := widget.Button("Save", view.Bg(core.Accent))
	fmt.Print(view.Dump(b))
	// Output:
	// flow padL=10 padR=10 bg=accent control=button lighten=true
	//   textB text="Save" grow=1 fg=chip-text ellipsis=true center=true
}
