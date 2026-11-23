package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Right-click menus for tabs, table rows and text widgets; the response's is in ui/response.
// The tree's own menu is in ui/sidebar/menu.go. Every item does what its button
// or shortcut does, through the same code.

var menuSeparator = MenuItem{Separator: true}

// ---- request tabs ------------------------------------------------------------

func (a *Component) tabMenuItems(index int) []MenuItem {
	t := a.widgets[a.tabs.Tabs()[index]]
	return []MenuItem{
		{Label: "Close", Run: func() { a.closeTab(index) }},
		{Label: "Close others", Disabled: len(a.tabs.Tabs()) == 1, Run: func() { a.closeOthers(index) }},
		{Label: "Close tabs to the right", Disabled: index == len(a.tabs.Tabs())-1, Run: func() { a.closeRightOf(index) }},
		menuSeparator,
		{Label: "Duplicate", Run: func() { a.duplicateTab(index) }},
		{Label: "Copy URL", Run: func() { a.copyRequestURL(t.Req) }},
		{Label: "Cookies...", Run: func() { a.editor.Auth.ShowCookies(t.Tab) }},
	}
}

// closeOthers closes every tab but the tab at index, which becomes the active one.
func (a *Component) closeOthers(index int) { a.updateTabs(tabs.CloseOthers{Index: index}) }

func (a *Component) closeRightOf(index int) { a.updateTabs(tabs.CloseRightOf{Index: index}) }

// duplicateTab copies the tab's request next to the original in the
// tree (or into the root when it is not in the tree) and opens the copy.
func (a *Component) duplicateTab(index int) {
	req := a.tabs.Tabs()[index].Req
	owner, ok := a.entryFor(req)
	if !ok { // nothing is open: the copy is a tab of its own
		clone := copyOf(req)
		a.openRequest(&clone)
		return
	}
	cp := owner.Collection.Folder.Duplicate(request.Node{Req: req}).Req
	if cp == nil {
		cp = owner.Collection.Folder.AddRequest(copyOf(req))
	}
	a.openRequest(cp)
}

// copyOf is a clone of req named as a copy.
func copyOf(req *request.Request) request.Request {
	clone := req.Clone()
	clone.Name += " copy"
	return clone
}
