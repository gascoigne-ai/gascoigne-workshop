package collections

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/openset"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// withEnvs is a collection with environments "dev" and "prod".
func withEnvs(name string) *request.Collection {
	col := newCollection(name)
	col.Envs = []request.Environment{{Name: "dev"}, {Name: "prod"}}
	return col
}

func TestRememberedListsOpenEntriesThenUnreadableOnes(t *testing.T) {
	first, second := withEnvs("a"), newCollection("b")
	set := Update(Set{}, Open{Collection: first, Dir: "/w/a"})
	set = Update(set, Open{Collection: second, Dir: "/w/b"})
	set = Update(set, ChooseEnv{Collection: first, Index: 1})
	unreadable := []openset.OpenCollection{
		{Folder: "/w/gone", Environment: "dev"},
		{Folder: "/w/b", Environment: "stale"}, // open now, so not kept twice
	}
	want := []openset.OpenCollection{
		{Folder: "/w/a", Environment: "prod"},
		{Folder: "/w/b"},
		{Folder: "/w/gone", Environment: "dev"},
	}
	if got := Remembered(set, unreadable); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestRememberedKeepsAnUnsavedCollectionAsAnEmptyFolder(t *testing.T) {
	set := Update(Set{}, Open{Collection: newCollection("sample")})
	want := []openset.OpenCollection{{}}
	if got := Remembered(set, nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWithEnvironmentsChangesOnlyEntriesNamingAnOpenCollection(t *testing.T) {
	col := withEnvs("a")
	set := Update(Set{}, Open{Collection: col, Dir: "/w/a"})
	set = Update(set, ChooseEnv{Collection: col, Index: 0})
	remembered := []openset.OpenCollection{
		{Folder: "/w/a", Environment: "prod"},
		{Folder: "/w/elsewhere", Environment: "prod"},
	}
	want := []openset.OpenCollection{
		{Folder: "/w/a", Environment: "dev"},
		{Folder: "/w/elsewhere", Environment: "prod"},
	}
	got, changed := WithEnvironments(set, remembered)
	if !changed || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v changed=%v, want %+v", got, changed, want)
	}
	if remembered[0].Environment != "prod" {
		t.Fatal("the list given was changed in place")
	}
	if _, changed := WithEnvironments(set, got); changed {
		t.Fatal("an up to date list reports a change")
	}
}
