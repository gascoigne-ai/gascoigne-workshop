package collections

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

func newCollection(name string) *request.Collection {
	return &request.Collection{Folder: request.Folder{Name: name}}
}

func names(set Set) []string {
	var out []string
	for _, entry := range set.Entries {
		out = append(out, entry.Collection.Name)
	}
	return out
}

func sameNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// opened is a set holding a, b and c, in that order, each in its own folder.
func opened() (Set, *request.Collection, *request.Collection, *request.Collection) {
	first, second, third := newCollection("a"), newCollection("b"), newCollection("c")
	var set Set
	set = Update(set, Open{Collection: first, Dir: "/w/a"})
	set = Update(set, Open{Collection: second, Dir: "/w/b"})
	set = Update(set, Open{Collection: third, Dir: "/w/c"})
	return set, first, second, third
}

func TestOpenAppendsInTheOrderOpened(t *testing.T) {
	set, _, _, _ := opened()
	if got := names(set); !sameNames(got, []string{"a", "b", "c"}) {
		t.Fatalf("order = %v", got)
	}
}

func TestOpenStartsWithNoEnvironment(t *testing.T) {
	col := newCollection("a")
	col.Envs = []request.Environment{{Name: "Dev"}}
	set := Update(Set{}, Open{Collection: col, Dir: "/w/a"})
	if env := set.Entries[0].Environment(); env != nil {
		t.Fatalf("a fresh collection has %q active, want none", env.Name)
	}
}

func TestOpenTheSameFolderTwiceKeepsTheOpenOne(t *testing.T) {
	set, first, _, _ := opened()
	set = Update(set, Open{Collection: newCollection("a again"), Dir: "/w/x/../a/"})
	if got := names(set); !sameNames(got, []string{"a", "b", "c"}) {
		t.Fatalf("order = %v", got)
	}
	entry, ok := set.ByDir("/w/a")
	if !ok || entry.Collection != first {
		t.Fatalf("ByDir found %v, %v; want the first one", entry.Collection, ok)
	}
}

func TestOpenTheSameCollectionTwiceKeepsOne(t *testing.T) {
	col := newCollection("unsaved")
	set := Update(Set{}, Open{Collection: col})
	set = Update(set, Open{Collection: col})
	if len(set.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(set.Entries))
	}
}

func TestUnsavedCollectionsAreAllKept(t *testing.T) {
	set := Update(Set{}, Open{Collection: newCollection("x")})
	set = Update(set, Open{Collection: newCollection("y")})
	if got := names(set); !sameNames(got, []string{"x", "y"}) {
		t.Fatalf("order = %v", got)
	}
	if _, ok := set.ByDir(""); ok {
		t.Fatal("ByDir(\"\") found an unsaved collection")
	}
}

func TestUpdateLeavesTheGivenSetAlone(t *testing.T) {
	set, first, _, _ := opened()
	_ = Update(set, Close{Collection: first})
	_ = Update(set, ChooseEnv{Collection: first, Index: 0})
	if len(set.Entries) != 3 || set.Entries[0].Env != -1 {
		t.Fatalf("the given set changed: %+v", set.Entries)
	}
}

func TestClose(t *testing.T) {
	set, first, second, third := opened()
	cases := []struct {
		name   string
		closed *request.Collection
		want   []string
	}{
		{"the first", first, []string{"b", "c"}},
		{"the middle", second, []string{"a", "c"}},
		{"the last", third, []string{"a", "b"}},
		{"one not open", newCollection("z"), []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		got := names(Update(set, Close{Collection: tc.closed}))
		if !sameNames(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestClosingTheLastLeavesNone(t *testing.T) {
	col := newCollection("a")
	set := Update(Update(Set{}, Open{Collection: col}), Close{Collection: col})
	if len(set.Entries) != 0 {
		t.Fatalf("entries = %v", names(set))
	}
}

func TestMove(t *testing.T) {
	set, first, second, third := opened()
	cases := []struct {
		name         string
		moved, place *request.Collection
		want         []string
	}{
		{"before the first", third, first, []string{"c", "a", "b"}},
		{"before a later one", first, third, []string{"b", "a", "c"}},
		{"to the end", first, nil, []string{"b", "c", "a"}},
		{"before itself", second, second, []string{"a", "b", "c"}},
		{"one not open", newCollection("z"), first, []string{"a", "b", "c"}},
		{"before one not open", first, newCollection("z"), []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		got := names(Update(set, Move{Collection: tc.moved, Before: tc.place}))
		if !sameNames(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestShift(t *testing.T) {
	set, first, second, third := opened()
	cases := []struct {
		name  string
		moved *request.Collection
		by    int
		want  []string
	}{
		{"middle up", second, -1, []string{"b", "a", "c"}},
		{"middle down", second, 1, []string{"a", "c", "b"}},
		{"first down", first, 1, []string{"b", "a", "c"}},
		{"last up", third, -1, []string{"a", "c", "b"}},
		{"first up stays", first, -1, []string{"a", "b", "c"}},
		{"last down stays", third, 1, []string{"a", "b", "c"}},
		{"one not open stays", newCollection("z"), -1, []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		got := names(Update(set, Shift{Collection: tc.moved, By: tc.by}))
		if !sameNames(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMoveKeepsWhatAnEntryHolds(t *testing.T) {
	set, first, _, _ := opened()
	set = Update(set, ChooseEnv{Collection: first, Index: -1})
	set = Update(set, Move{Collection: first})
	entry, _ := set.Owner(request.Of(&first.Folder))
	if entry.Dir != "/w/a" || entry.Collection != first {
		t.Fatalf("moved entry = %+v", entry)
	}
}

func TestLocatedGivesAnUnsavedCollectionItsFolder(t *testing.T) {
	col := newCollection("new")
	set := Update(Set{}, Open{Collection: col})
	set = Update(set, Located{Collection: col, Dir: "/w/new/"})
	entry, ok := set.ByDir("/w/new")
	if !ok || entry.Collection != col {
		t.Fatalf("ByDir = %+v, %v", entry, ok)
	}
}

// Two open collections in one folder would overwrite each other's files:
// the first save of an unsaved collection and a rename both have to be
// refused a folder another open collection holds.
func TestLocatedRefusesAFolderAnotherCollectionHolds(t *testing.T) {
	set, first, second, _ := opened()
	if !set.Taken("/w/b", first) {
		t.Fatal("Taken(/w/b) for a = false, want true")
	}
	if set.Taken("/w/b", second) {
		t.Fatal("Taken(/w/b) for b itself = true, want false")
	}
	if set.Taken("", first) {
		t.Fatal("no folder is taken")
	}
	set = Update(set, Located{Collection: first, Dir: "/w/b"})
	entry, _ := set.ByCollection(first)
	if entry.Dir != "/w/a" {
		t.Fatalf("a moved into b's folder: %q", entry.Dir)
	}
}

func TestChooseEnv(t *testing.T) {
	col := newCollection("a")
	col.Envs = []request.Environment{{Name: "Dev"}, {Name: "Prod"}}
	set := Update(Set{}, Open{Collection: col})
	cases := []struct {
		name  string
		index int
		want  string
	}{
		{"one in range", 1, "Prod"},
		{"none", -1, ""},
		{"past the end is none", 2, ""},
	}
	for _, tc := range cases {
		entry := Update(set, ChooseEnv{Collection: col, Index: tc.index}).Entries[0]
		got := ""
		if env := entry.Environment(); env != nil {
			got = env.Name
		}
		if got != tc.want {
			t.Errorf("%s: active = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestOwnerFindsTheCollectionOfANode(t *testing.T) {
	set, first, second, _ := opened()
	folder := second.AddFolder("nested")
	req := folder.AddRequest(request.Request{Name: "deep"})
	cases := []struct {
		name string
		node request.Node
		want *request.Collection
	}{
		{"its root folder", request.Of(&first.Folder), first},
		{"a folder", request.Of(folder), second},
		{"a request in a folder", request.Of(req), second},
	}
	for _, tc := range cases {
		entry, ok := set.Owner(tc.node)
		if !ok || entry.Collection != tc.want {
			t.Errorf("%s: owner = %v, %v; want %q", tc.name, entry.Collection, ok, tc.want.Name)
		}
	}
	if _, ok := set.Owner(request.Of(&request.Request{Name: "loose"})); ok {
		t.Error("a request in no open collection has an owner")
	}
	if _, ok := set.Owner(request.Node{}); ok {
		t.Error("no node has an owner")
	}
}

// Variables never leak across collections: a request is sent with its own
// collection's variables and that collection's environment only, and a
// request of no open collection gets none rather than the first one's.
func TestVarsForUsesOnlyTheOwnersVariables(t *testing.T) {
	alpha, beta := newCollection("alpha"), newCollection("beta")
	alpha.Vars = []request.KV{{Name: "host", Value: "alpha.local", Enabled: true}}
	alpha.Envs = []request.Environment{{Name: "Dev", Vars: []request.KV{{Name: "token", Value: "alpha-dev", Enabled: true}}}}
	beta.Vars = []request.KV{{Name: "host", Value: "beta.local", Enabled: true}}
	beta.Envs = []request.Environment{{Name: "Dev", Vars: []request.KV{{Name: "token", Value: "beta-dev", Enabled: true}}}}
	alphaReq := alpha.AddRequest(request.Request{Name: "a"})
	betaReq := beta.AddRequest(request.Request{Name: "b"})

	set := Update(Set{}, Open{Collection: alpha, Dir: "/w/alpha"})
	set = Update(set, Open{Collection: beta, Dir: "/w/beta"})
	set = Update(set, ChooseEnv{Collection: alpha, Index: 0})

	cases := []struct {
		name string
		req  *request.Request
		want map[string]string
	}{
		{"alpha with its environment", alphaReq, map[string]string{"host": "alpha.local", "token": "alpha-dev"}},
		{"beta without one", betaReq, map[string]string{"host": "beta.local"}},
	}
	for _, tc := range cases {
		vars, ok := set.VarsFor(tc.req)
		got := map[string]string{}
		for _, kv := range vars {
			got[kv.Name] = kv.Value
		}
		if !ok || len(got) != len(tc.want) {
			t.Errorf("%s: vars = %v, %v; want %v", tc.name, got, ok, tc.want)
			continue
		}
		for name, value := range tc.want {
			if got[name] != value {
				t.Errorf("%s: %s = %q, want %q", tc.name, name, got[name], value)
			}
		}
	}
	if vars, ok := set.VarsFor(&request.Request{Name: "loose"}); ok || vars != nil {
		t.Errorf("a request of no open collection got %v, %v", vars, ok)
	}
}

func TestByCollectionFindsTheEntryOfAnOpenCollection(t *testing.T) {
	set, first, _, _ := opened()
	entry, ok := set.ByCollection(first)
	if !ok || entry.Collection != first {
		t.Fatalf("entry %+v, %v; want the first collection's", entry, ok)
	}
	if _, ok := set.ByCollection(newCollection("closed")); ok {
		t.Error("a collection that is not open has an entry")
	}
}
