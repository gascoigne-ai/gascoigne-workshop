package tabs

import (
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// listOf is a list of tabs named by the letters of names, the tab at active
// being the active one and none being dragged.
func listOf(names string, active int) List {
	list := List{Active: active, Drag: -1}
	for _, name := range strings.Split(names, "") {
		list.Tabs = append(list.Tabs, New(&model.Request{Name: name}))
	}
	return list
}

// names are the tab names in order, "" for an empty list.
func names(list List) string {
	var out []string
	for _, tab := range list.Tabs {
		out = append(out, tab.Req.Name)
	}
	return strings.Join(out, "")
}

// activeName is the name of the active tab.
func activeName(list List) string { return list.Tabs[list.Active].Req.Name }

func TestUpdateOpenFocusesAnOpenRequestOrAddsATab(t *testing.T) {
	list := listOf("abc", 2)
	existing := list.Tabs[0].Req
	fresh := &model.Request{Name: "d"}

	cases := []struct {
		name       string
		req        *model.Request
		wantNames  string
		wantActive string
	}{
		{"an open request is focused, not opened twice", existing, "abc", "a"},
		{"a new request opens last and is focused", fresh, "abcd", "d"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Update(list, Open{Req: c.req})
			if names(got) != c.wantNames || activeName(got) != c.wantActive {
				t.Fatalf("got tabs %q active %q, want %q active %q", names(got), activeName(got), c.wantNames, c.wantActive)
			}
		})
	}
}

func TestUpdateCloseKeepsTheActiveTabWhereItIs(t *testing.T) {
	cases := []struct {
		name       string
		list       List
		index      int
		wantNames  string
		wantActive string
	}{
		{"closing a tab before the active one", listOf("abcd", 2), 0, "bcd", "c"},
		{"closing the active tab moves to its neighbour", listOf("abcd", 1), 1, "acd", "c"},
		{"closing the active last tab moves to the new last", listOf("abc", 2), 2, "ab", "b"},
		{"closing a tab after the active one", listOf("abc", 0), 2, "ab", "a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Update(c.list, Close{Index: c.index})
			if names(got) != c.wantNames || activeName(got) != c.wantActive {
				t.Fatalf("got tabs %q active %q, want %q active %q", names(got), activeName(got), c.wantNames, c.wantActive)
			}
		})
	}
}

func TestUpdateClosingTheLastTabLeavesABlankRequest(t *testing.T) {
	got := Update(listOf("a", 0), Close{Index: 0})

	if len(got.Tabs) != 1 || got.Active != 0 {
		t.Fatalf("got %d tabs, active %d; want one blank tab", len(got.Tabs), got.Active)
	}
	req := got.Tabs[0].Req
	if req.Name != "New Request" || req.Protocol != "rest" || req.Method != "GET" {
		t.Fatalf("blank request is %+v", *req)
	}
}

func TestUpdateCloseOthersKeepsOnlyTheChosenTabAndActivatesIt(t *testing.T) {
	got := Update(listOf("abcd", 0), CloseOthers{Index: 2})

	if names(got) != "c" || got.Active != 0 {
		t.Fatalf("got tabs %q active %d, want c active 0", names(got), got.Active)
	}
}

func TestUpdateCloseRightOfClosesOnlyLaterTabs(t *testing.T) {
	cases := []struct {
		name       string
		list       List
		wantActive string
	}{
		{"an active tab to the right falls back to the chosen one", listOf("abcd", 3), "b"},
		{"an active tab to the left stays", listOf("abcd", 0), "a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Update(c.list, CloseRightOf{Index: 1})
			if names(got) != "ab" || activeName(got) != c.wantActive {
				t.Fatalf("got tabs %q active %q, want ab active %q", names(got), activeName(got), c.wantActive)
			}
		})
	}
}

func TestUpdateActivateAndStep(t *testing.T) {
	cases := []struct {
		name string
		list List
		msg  Msg
		want int
	}{
		{"activate picks the tab", listOf("abc", 0), Activate{Index: 2}, 2},
		{"step forward", listOf("abc", 0), Step{By: 1}, 1},
		{"step forward wraps to the first", listOf("abc", 2), Step{By: 1}, 0},
		{"step back wraps to the last", listOf("abc", 0), Step{By: -1}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Update(c.list, c.msg); got.Active != c.want {
				t.Fatalf("active %d, want %d", got.Active, c.want)
			}
		})
	}
}

func TestUpdateDraggingATabMovesItAndKeepsTheActiveTabActive(t *testing.T) {
	list := Update(listOf("abc", 0), DragBegin{Index: 0})
	if list.Drag != 0 || list.Active != 0 {
		t.Fatalf("after DragBegin: drag %d active %d, want 0 and 0", list.Drag, list.Active)
	}

	list = Update(list, DragOver{Index: 2})
	if names(list) != "bca" || list.Drag != 2 || activeName(list) != "a" {
		t.Fatalf("after DragOver: tabs %q drag %d active %q, want bca 2 a", names(list), list.Drag, activeName(list))
	}

	list = Update(list, DragEnd{})
	if list.Drag != -1 {
		t.Fatalf("after DragEnd: drag %d, want -1", list.Drag)
	}
}

func TestUpdateDragOverDoesNothingWithoutADragOrOverTheDraggedTab(t *testing.T) {
	cases := []struct {
		name string
		list List
		over int
	}{
		{"no drag in progress", listOf("abc", 0), 2},
		{"pointer over the dragged tab itself", Update(listOf("abc", 0), DragBegin{Index: 1}), 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Update(c.list, DragOver{Index: c.over}); names(got) != "abc" {
				t.Fatalf("tabs %q, want abc", names(got))
			}
		})
	}
}

func TestUpdateDoesNotChangeTheListItWasGiven(t *testing.T) {
	list := listOf("abc", 1)
	before := slices.Clone(list.Tabs)

	Update(list, Close{Index: 0})
	Update(list, DragOver{Index: 2})

	if !slices.Equal(list.Tabs, before) || list.Active != 1 {
		t.Fatalf("the input list changed: %q active %d", names(list), list.Active)
	}
}

func TestUpdateIgnoresAnIndexOfATabThatIsGone(t *testing.T) {
	// A menu can outlive its tab; its stale index must not panic.
	list := listOf("ab", 0)
	for _, msg := range []Msg{Close{Index: 5}, CloseOthers{Index: 5}, CloseRightOf{Index: 5}, Activate{Index: 5}, DragBegin{Index: 5}, Close{Index: -1}} {
		if got := Update(list, msg); names(got) != "ab" || got.Active != 0 {
			t.Fatalf("%T with a stale index changed the list to %q active %d", msg, names(got), got.Active)
		}
	}
}

func TestUpdateResetEmptiesTheListAndStepOnAnEmptyListIsSafe(t *testing.T) {
	list := Update(listOf("abc", 2), DragBegin{Index: 1})

	got := Update(list, Reset{})
	if len(got.Tabs) != 0 || got.Active != 0 || got.Drag != -1 {
		t.Fatalf("after Reset: %d tabs, active %d, drag %d; want none, 0, -1", len(got.Tabs), got.Active, got.Drag)
	}

	if after := Update(got, Step{By: 1}); after.Active != 0 {
		t.Fatalf("Step on an empty list moved the active tab to %d", after.Active)
	}
}
