package tabs

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// setOf is a set of tabs named by the letters of names, the tab at active current.
func setOf(names string, active int) *Set {
	set := NewSet()
	for _, name := range strings.Split(names, "") {
		set.Open(&request.Request{Name: name})
	}
	set.Apply(Activate{Index: active})
	return set
}

func setNames(set *Set) string {
	var out []string
	for _, tab := range set.Tabs() {
		out = append(out, tab.Req.Name)
	}
	return strings.Join(out, "")
}

func TestSetOpenAddsATabAndMakesItCurrent(t *testing.T) {
	set := NewSet()
	if set.Current() != nil {
		t.Fatalf("an empty set has a current tab")
	}
	req := &request.Request{Name: "a"}
	set.Open(req)
	if len(set.Tabs()) != 1 || set.Current() == nil || set.Current().Req != req {
		t.Fatalf("got %q current %v", setNames(set), set.Current())
	}
}

func TestSetOpenOfAnOpenRequestFocusesItsTab(t *testing.T) {
	set := setOf("abc", 2)
	first := set.Tabs()[0]
	set.Open(first.Req)
	if setNames(set) != "abc" || set.Current() != first || set.Active() != 0 {
		t.Fatalf("got %q active %d", setNames(set), set.Active())
	}
}

func TestSetCloseCurrentPicksItsNeighbour(t *testing.T) {
	cases := []struct {
		name   string
		tabs   string
		active int
		close  int
		want   string
	}{
		{"middle of four", "abcd", 1, 1, "c"},
		{"last of three", "abc", 2, 2, "b"},
		{"first of four with c active", "abcd", 2, 0, "c"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			set := setOf(c.tabs, c.active)
			set.Close(c.close)
			if got := set.Current().Req.Name; got != c.want {
				t.Fatalf("current %q, want %q (tabs %q)", got, c.want, setNames(set))
			}
		})
	}
}

func TestSetClosingTheLastTabLeavesABlankCurrentTab(t *testing.T) {
	set := setOf("a", 0)
	set.Close(0)
	if len(set.Tabs()) != 1 || set.Current() == nil || set.Current().Req.Name != "New Request" {
		t.Fatalf("got %q", setNames(set))
	}
}

func TestNewSetDragsNothing(t *testing.T) {
	if got := NewSet().Drag(); got != -1 {
		t.Fatalf("drag %d, want -1", got)
	}
}

func TestSetApplyMatchesUpdate(t *testing.T) {
	cases := []struct {
		name string
		msg  Msg
	}{
		{"activate", Activate{Index: 0}},
		{"step", Step{By: 1}},
		{"close others", CloseOthers{Index: 1}},
		{"close right of", CloseRightOf{Index: 0}},
		{"drag begin", DragBegin{Index: 2}},
		{"drag over", DragOver{Index: 0}},
		{"reset", Reset{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			set := setOf("abc", 2)
			set.Apply(DragBegin{Index: 2})
			want := Update(set.list, c.msg)
			set.Apply(c.msg)
			if !reflect.DeepEqual(set.list, want) {
				t.Fatalf("got %+v, want %+v", set.list, want)
			}
		})
	}
}

func TestSetCurrentIsAlwaysAMember(t *testing.T) {
	set := setOf("abcd", 3)
	msgs := []Msg{Close{Index: 3}, Step{By: 1}, CloseRightOf{Index: 0}, DragBegin{Index: 0}, CloseOthers{Index: 0}, Close{Index: 0}, Reset{}}
	for _, msg := range msgs {
		set.Apply(msg)
		if set.Current() == nil {
			continue
		}
		member := false
		for _, tab := range set.Tabs() {
			member = member || tab == set.Current()
		}
		if !member {
			t.Fatalf("after %T the current tab is not a member", msg)
		}
	}
}
