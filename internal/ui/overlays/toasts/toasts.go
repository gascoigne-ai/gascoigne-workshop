// Package toasts is the stack of short messages in the window's bottom-right
// corner: an info toast fades after a few seconds, an error stays until it
// is clicked.
//
// Prune is not a Tick. Its result feeds the frame's "something moved"
// decision, which is taken before the view is built, while a Tick runs after
// it and only marks the tree stale. Revisit when the frame loop takes both
// kinds of change from one return value.
package toasts

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// kinds is how each severity looks as a toast; the toolkit does not know
// the severity levels.
var kinds = map[severity.Level]ToastKind{
	severity.Info:  ToastInfo,
	severity.Error: ToastError,
}

var template = markup.Must("toasts.vml", source)

// Component is the stack of toasts shown.
type Component struct {
	deps  Deps
	stack ToastStack
}

// New is the component with no toast.
func New(deps Deps) *Component { return &Component{deps: deps} }

// Show shows a message. The tree is always stale after it, the screen needs
// a draw only when the stack changed.
func (c *Component) Show(text string, level severity.Level) {
	c.deps.Window.Stale()
	if c.stack.Show(text, kinds[level], c.deps.Clock.Now()) {
		c.deps.Window.Redraw()
	}
}

// Prune drops the info toasts whose time is up and reports whether any was.
func (c *Component) Prune() bool { return c.stack.Prune(c.deps.Clock.Now()) }

// Stack is the toasts shown, oldest first.
func (c *Component) Stack() ToastStack { return c.stack }

// View is the stack of toasts above the status bar.
func (c *Component) View() Node {
	width, height := c.deps.Window.Size()
	return template.MustBuild(markup.Scope{
		"stack":   &c.stack,
		"width":   width,
		"height":  height,
		"statusH": sizes.StatusH,
	})
}

// Template, parsed once above.
const source = `<toasts stack="stack" width="{{width}}" height="{{height}}" bottom-inset="{{statusH}}"/>
`
