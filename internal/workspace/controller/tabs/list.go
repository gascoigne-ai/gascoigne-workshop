package tabs

import (
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// List is the open tabs in order, the active one and the one being dragged.
type List struct {
	Tabs   []*Tab
	Active int
	Drag   int // index of the tab being dragged, -1 for none
}

// Msg is a change to a List.
type Msg interface{ listMsg() }

// Open focuses the tab of Req, opening one when there is none.
type Open struct{ Req *model.Request }

// Close closes tab Index; closing the last tab leaves a blank request.
type Close struct{ Index int }

// CloseOthers closes every tab but Index, which becomes the active one.
type CloseOthers struct{ Index int }

// CloseRightOf closes the tabs after Index.
type CloseRightOf struct{ Index int }

// Activate makes tab Index the active one.
type Activate struct{ Index int }

// Step activates the tab By places away, wrapping at the ends.
type Step struct{ By int }

// DragBegin starts dragging tab Index and activates it.
type DragBegin struct{ Index int }

// DragOver moves the dragged tab to where the pointer is, over tab Index.
type DragOver struct{ Index int }

// DragEnd ends the drag.
type DragEnd struct{}

func (Open) listMsg()         {}
func (Close) listMsg()        {}
func (CloseOthers) listMsg()  {}
func (CloseRightOf) listMsg() {}
func (Activate) listMsg()     {}
func (Step) listMsg()         {}
func (DragBegin) listMsg()    {}
func (DragOver) listMsg()     {}
func (DragEnd) listMsg()      {}

// Update returns the list after msg. The list it is given is not changed.
// A message that names a tab that is no longer there, such as a menu that
// outlived its tab, leaves the list as it is.
func Update(list List, msg Msg) List {
	list.Tabs = slices.Clone(list.Tabs)
	switch msg := msg.(type) {
	case Open:
		return open(list, msg.Req)
	case Close:
		return closeAt(list, msg.Index)
	case CloseOthers:
		return closeOthers(list, msg.Index)
	case CloseRightOf:
		for index := len(list.Tabs) - 1; index > msg.Index && list.has(msg.Index); index-- {
			list = closeAt(list, index)
		}
	case Activate:
		if list.has(msg.Index) {
			list.Active = msg.Index
		}
	case Step:
		list.Active = Next(list.Active, len(list.Tabs), msg.By)
	case DragBegin:
		if list.has(msg.Index) {
			list.Active, list.Drag = msg.Index, msg.Index
		}
	case DragOver:
		if list.Drag >= 0 && list.Drag != msg.Index && list.has(msg.Index) {
			list = move(list, list.Drag, msg.Index)
		}
	case DragEnd:
		list.Drag = -1
	}
	return list
}

func (list List) has(index int) bool { return index >= 0 && index < len(list.Tabs) }

func open(list List, req *model.Request) List {
	for index, tab := range list.Tabs {
		if tab.Req == req {
			list.Active = index
			return list
		}
	}
	list.Tabs = append(list.Tabs, New(req))
	list.Active = len(list.Tabs) - 1
	return list
}

func closeAt(list List, index int) List {
	if !list.has(index) {
		return list
	}
	list.Tabs = slices.Delete(list.Tabs, index, index+1)
	if len(list.Tabs) == 0 {
		list.Tabs = append(list.Tabs, New(&model.Request{Name: "New Request", Protocol: "rest", Method: "GET"}))
	}
	list.Active = AfterClose(list.Active, index, len(list.Tabs))
	list.Drag = -1
	return list
}

func closeOthers(list List, index int) List {
	if !list.has(index) {
		return list
	}
	list.Tabs = []*Tab{list.Tabs[index]}
	list.Active, list.Drag = 0, -1
	return list
}

// move puts the tab at from at place to; the active tab stays the same tab.
func move(list List, from, to int) List {
	list.Active = AfterMove(list.Active, from, to)
	tab := list.Tabs[from]
	list.Tabs = slices.Insert(slices.Delete(list.Tabs, from, from+1), to, tab)
	list.Drag = to
	return list
}
