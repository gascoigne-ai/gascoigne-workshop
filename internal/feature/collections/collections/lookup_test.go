package collections

import (
	"slices"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// sample is a collection with a request at the top and one in a nested
// folder: Top, Docs/Intro, Docs/Deep/Leaf.
func sample(name string) (col *request.Collection, top, intro, leaf *request.Request, docs, deep *request.Folder) {
	top = &request.Request{Name: "Top"}
	intro = &request.Request{Name: "Intro"}
	leaf = &request.Request{Name: "Leaf"}
	deep = &request.Folder{Name: "Deep", Requests: []*request.Request{leaf}}
	docs = &request.Folder{Name: "Docs", Requests: []*request.Request{intro}, Folders: []*request.Folder{deep}}
	col = &request.Collection{Folder: request.Folder{Name: name, Requests: []*request.Request{top}, Folders: []*request.Folder{docs}}}
	return col, top, intro, leaf, docs, deep
}

func TestPathOfNamesTheOwnerAndThePath(t *testing.T) {
	col, top, _, leaf, _, _ := sample("a")
	set := Update(Set{}, Open{Collection: col, Dir: "/w/a"})
	tests := []struct {
		name     string
		req      *request.Request
		wantPath string
		wantOK   bool
	}{
		{"top", top, "Top", true},
		{"nested", leaf, "Docs/Deep/Leaf", true},
		{"not open", &request.Request{Name: "Top"}, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry, path, ok := set.PathOf(test.req)
			if path != test.wantPath || ok != test.wantOK || (ok && entry.Collection != col) {
				t.Fatalf("got %v, %q, %v; want %q, %v", entry.Dir, path, ok, test.wantPath, test.wantOK)
			}
		})
	}
}

func TestDirsAreInTreeOrder(t *testing.T) {
	set, _, _, _ := opened()
	set = Update(set, Open{Collection: newCollection("d")})
	want := []string{"/w/a", "/w/b", "/w/c", ""}
	if got := set.Dirs(); !slices.Equal(got, want) {
		t.Fatalf("dirs = %q, want %q", got, want)
	}
}

func TestExpandedPathsListsTheOpenFolders(t *testing.T) {
	col, _, _, _, docs, deep := sample("a")
	set := Update(Set{}, Open{Collection: col, Dir: "/w/a"})
	entry := set.Entries[0]
	tests := []struct {
		name string
		open map[*request.Folder]bool
		want []string
	}{
		{"none", nil, nil},
		{"one", map[*request.Folder]bool{docs: true}, []string{"Docs"}},
		{"nested after its parent", map[*request.Folder]bool{deep: true, docs: true}, []string{"Docs", "Docs/Deep"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := set.ExpandedPaths(entry, func(folder *request.Folder) bool { return test.open[folder] })
			if !slices.Equal(got, test.want) {
				t.Fatalf("paths = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFoldersAtFindsTheRememberedPaths(t *testing.T) {
	col, _, _, _, docs, deep := sample("a")
	tests := []struct {
		name  string
		col   *request.Collection
		paths []string
		want  []*request.Folder
	}{
		{"known paths", col, []string{"Docs/Deep", "Docs"}, []*request.Folder{deep, docs}},
		{"a path that is gone", col, []string{"Gone", "Docs"}, []*request.Folder{docs}},
		{"no collection", nil, []string{"Docs"}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := (Set{}).FoldersAt(test.col, test.paths); !slices.Equal(got, test.want) {
				t.Fatalf("folders = %v, want %v", got, test.want)
			}
		})
	}
}

func TestHistoryKeyJoinsFolderAndPath(t *testing.T) {
	saved, _, _, savedLeaf, _, _ := sample("a")
	unsaved, unsavedTop, _, _, _, _ := sample("b")
	set := Update(Set{}, Open{Collection: saved, Dir: "/w/a"})
	set = Update(set, Open{Collection: unsaved})
	tests := []struct {
		name string
		req  *request.Request
		want string
	}{
		{"saved", savedLeaf, "/w/a#Docs/Deep/Leaf"},
		{"unsaved", unsavedTop, "Top"},
		{"not open", &request.Request{Name: "Top"}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := set.HistoryKey(test.req); got != test.want {
				t.Fatalf("key = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLegacyHistoryKeyOnlyForTheFirstSavedCollection(t *testing.T) {
	first, firstTop, _, _, _, _ := sample("a")
	second, secondTop, _, _, _, _ := sample("b")
	unsaved, unsavedTop, _, _, _, _ := sample("c")
	set := Update(Set{}, Open{Collection: first, Dir: "/w/a"})
	set = Update(set, Open{Collection: second, Dir: "/w/b"})
	set = Update(set, Open{Collection: unsaved})
	tests := []struct {
		name string
		req  *request.Request
		want string
	}{
		{"first saved", firstTop, "Top"},
		{"second saved", secondTop, ""},
		{"unsaved", unsavedTop, ""},
		{"not open", &request.Request{Name: "Top"}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := set.LegacyHistoryKey(test.req); got != test.want {
				t.Fatalf("key = %q, want %q", got, test.want)
			}
		})
	}
}
