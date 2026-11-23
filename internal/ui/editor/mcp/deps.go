package mcp

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the component asks of the application, each a narrow
// interface the host implements.
type Deps struct {
	Tabs     Tabs
	Bus      Bus
	Composer Composer
	Dialogs  Dialogs
	Looks    Looks
}

// Tabs says which tabs are still open.
type Tabs interface {
	IsOpen(tab *tabs.Tab) bool
}

// Bus moves work off the UI goroutine and back.
type Bus interface {
	// Run does work on its own goroutine and runs the answer it returns,
	// when it returns one, on the UI goroutine.
	Run(work func() (answer func()))
	// Dispatch runs run on the UI goroutine. It is safe from any goroutine.
	Dispatch(run func())
}

// Composer is the message box of a tab, where a tool call is composed.
type Composer interface {
	// SetComposer replaces the text of the composer of tab.
	SetComposer(tab *tabs.Tab, text string)
	// ComposeCall puts the call of tool, with the defaults of its schema,
	// into the composer of tab.
	ComposeCall(tab *tabs.Tab, tool request.MCPTool)
}

// Dialogs is the one dialog the window shows over everything.
type Dialogs interface {
	// Open shows state, with content between its text and its buttons when
	// content is not nil, in place of any open dialog. The returned func
	// closes that dialog; it does nothing when another one replaced it.
	Open(state widget.DialogState, content func(ctx *core.Ctx) view.Node) (close func())
	// Busy reports whether a dialog is open.
	Busy() bool
}

// Looks are the texts and lines the explorer shares with panes this package
// may not import.
type Looks interface {
	// StateText is what the explorer says of a connection in state.
	StateText(state tabs.ConnState) string
	// CodeLines are the colored lines of a JSON answer.
	CodeLines(body []byte) []widget.Line
	// RawLines are the plain lines of an answer that is not JSON.
	RawLines(body []byte) []widget.Line
}
