package tabs

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"

// Set is the owner of the open tabs and the current one. The invariant is
// that the current tab is nil or one of Tabs: Current is computed from the
// active index of the list, never stored beside it. A slice, because the
// order is the strip order and there are tens of tabs.
//
// A Set only keeps the tabs. What a tab owns elsewhere (its widgets, its
// send, its socket) is the caller's to sync after each change; the app does
// it in one place, App.updateTabs. A change callback from Set was rejected:
// it would run before the caller can tell which tabs left.
type Set struct{ list List }

// NewSet is a set with no tab open and none being dragged.
func NewSet() *Set { return &Set{list: List{Drag: -1}} }

// Tabs are the open tabs in the order of the strip.
func (set *Set) Tabs() []*Tab { return set.list.Tabs }

// Current is the active tab, nil before the first Open and after a Reset.
// Closing the last tab leaves a blank one, so it is not nil then.
func (set *Set) Current() *Tab {
	if !set.list.has(set.list.Active) {
		return nil
	}
	return set.list.Tabs[set.list.Active]
}

// Active is the index of the current tab, 0 when none is open.
func (set *Set) Active() int { return set.list.Active }

// Drag is the index of the tab being dragged, -1 for none.
func (set *Set) Drag() int { return set.list.Drag }

// Open focuses the tab of req, opening one when there is none.
func (set *Set) Open(req *request.Request) { set.Apply(Open{Req: req}) }

// Close closes tab index; closing the last tab leaves a blank request.
func (set *Set) Close(index int) { set.Apply(Close{Index: index}) }

// Apply changes the set the way Update changes a List.
func (set *Set) Apply(msg Msg) { set.list = Update(set.list, msg) }
