package mcp

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// OpenToolForm shows the form of tool; its answer fills the composer of
// tab. A tool with no properties is composed at once. The form lives in the
// closures of the dialog: the window owns the one open dialog, so there is
// no place for it among the states of the tabs.
func (c *Component) OpenToolForm(tab *tabs.Tab, tool request.MCPTool) {
	form := mcpform.New(tool)
	if form.Empty() {
		c.deps.Composer.ComposeCall(tab, tool)
		return
	}
	var closeDialog func()
	fill := func() { c.fill(tab, form, closeDialog) }
	closeDialog = c.deps.Dialogs.Open(widget.DialogState{
		Title: "Call " + tool.Name,
		Buttons: []widget.DialogButton{
			{Label: "Cancel", Run: func() { closeDialog() }},
			{Label: "Fill composer", Primary: true, Run: fill},
		},
	}, func(ctx *core.Ctx) view.Node { return form.View(ctx, submit(fill)) })
}

// fill puts the call the form describes into the composer of tab and closes
// the dialog, or keeps the form open and says what is wrong.
func (c *Component) fill(tab *tabs.Tab, form *mcpform.Form, closeDialog func()) {
	text, err := form.Command()
	if err != nil {
		form.Problem = err.Error()
		return
	}
	closeDialog()
	c.deps.Composer.SetComposer(tab, text)
}
