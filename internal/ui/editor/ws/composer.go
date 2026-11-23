package ws

import (
	"context"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/outbox"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// SetComposer replaces the composer's text of tab with text and focuses it.
func (c *Component) SetComposer(tab *tabs.Tab, text string) {
	composer := c.Composer(tab)
	*composer = widget.NewTextField(text)
	composer.Focused = true
}

// ComposeCall fills the composer of tab with a call of tool.
func (c *Component) ComposeCall(tab *tabs.Tab, tool request.MCPTool) {
	args := "{}"
	if tool.Schema.Type != "" {
		args = request.Skeleton(tool.Schema)
	}
	c.SetComposer(tab, "call "+tool.Name+" "+args)
}

// SendMessage sends the composer's text of tab and empties it. A text that
// cannot be sent is explained in the log and stays in the composer.
func (c *Component) SendMessage(tab *tabs.Tab) {
	text := c.Composer(tab).Text
	if tab.Session() == nil || text == "" {
		return
	}
	send, err := composed(tab.Session(), text)
	if err != nil {
		c.Note(tab, err.Error(), severity.Error)
		return
	}
	c.deps.Outbox.Queue(tab, send)
	c.SetComposer(tab, "")
}

// Ping sends a ping on the session of tab.
func (c *Component) Ping(tab *tabs.Tab) {
	if tab.Session() == nil {
		return
	}
	c.deps.Outbox.Queue(tab, tab.Session().Ping)
}

// composed is the send of the composer's text, ready to run; a text that
// cannot be sent (bad hex) is refused here, before anything is sent.
func composed(session request.Session, text string) (outbox.Send, error) {
	binarySender, canBinary := session.(request.BinarySender)
	if !canBinary {
		return func(ctx context.Context) error { return session.Send(ctx, text) }, nil
	}
	data, binary, err := send.BinaryMessage(text)
	if err != nil {
		return nil, err
	}
	if !binary {
		return func(ctx context.Context) error { return session.Send(ctx, text) }, nil
	}
	return func(ctx context.Context) error { return binarySender.SendBinary(ctx, data) }, nil
}

// toolMenu lists the tools the server announced; picking one asks the host
// for the tool's form, which fills the composer.
func (c *Component) toolMenu(tab *tabs.Tab) []widget.MenuItem {
	items := make([]widget.MenuItem, 0, len(tab.Tools))
	for _, tool := range tab.Tools {
		items = append(items, widget.MenuItem{Label: tool.Name, Run: func() { c.deps.Tools.OpenToolForm(tab, tool) }})
	}
	return items
}

// resourceMenu lists the announced resources; picking one composes a read.
func (c *Component) resourceMenu(tab *tabs.Tab) []widget.MenuItem {
	items := make([]widget.MenuItem, 0, len(tab.Resources))
	for _, resource := range tab.Resources {
		label := resource.URI
		if resource.Name != "" {
			label = resource.Name + "  " + resource.URI
		}
		items = append(items, widget.MenuItem{Label: label, Run: func() {
			c.SetComposer(tab, "read "+resource.URI)
		}})
	}
	return items
}

// promptMenu lists the announced prompts; picking one composes a request
// with its required arguments filled in.
func (c *Component) promptMenu(tab *tabs.Tab) []widget.MenuItem {
	items := make([]widget.MenuItem, 0, len(tab.Prompts))
	for _, prompt := range tab.Prompts {
		items = append(items, widget.MenuItem{Label: prompt.Name, Run: func() {
			c.SetComposer(tab, "prompt "+prompt.Name+" "+prompt.ArgsJSON())
		}})
	}
	return items
}

// formatMessage turns the composer's JSON into one line, so a pasted,
// indented message goes out as a single frame; text that is not JSON
// stays as typed and a toast says where it breaks.
func (c *Component) formatMessage(tab *tabs.Tab) {
	compact, err := format.CompactJSON(c.Composer(tab).Text)
	if err != nil {
		c.deps.Window.Toast(err.Error(), severity.Error)
		return
	}
	c.SetComposer(tab, compact)
}
