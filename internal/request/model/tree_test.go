package model

import (
	"errors"
	"reflect"
	"testing"
)

// root:  R
//
//	├─ A   (a1, a2, folder A1 (x))
//	├─ B   (b1)
//	└─ r1
func fixture() (root *Folder, a, a1f, b *Folder, a1, a2, x, b1, r1 *Request) {
	a1, a2, x, b1, r1 = &Request{Name: "a1"}, &Request{Name: "a2"}, &Request{Name: "x"}, &Request{Name: "b1"}, &Request{Name: "r1"}
	a1f = &Folder{Name: "A1", Requests: []*Request{x}}
	a = &Folder{Name: "A", Folders: []*Folder{a1f}, Requests: []*Request{a1, a2}}
	b = &Folder{Name: "B", Requests: []*Request{b1}}
	root = &Folder{Name: "R", Folders: []*Folder{a, b}, Requests: []*Request{r1}}
	return
}

func names(reqs []*Request) []string {
	var out []string
	for _, r := range reqs {
		out = append(out, r.Name)
	}
	return out
}

func folderNames(fs []*Folder) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Name)
	}
	return out
}

func TestUniqueNameCountsFoldersAndRequests(t *testing.T) {
	root, a, _, _, _, _, _, _, _ := fixture()
	tests := []struct {
		folder *Folder
		base   string
		want   string
	}{
		{root, "fresh", "fresh"},
		{root, "r1", "r1 2"},
		{root, "A", "A 2"}, // a folder name counts too
		{a, "a1", "a1 2"},
		{a, "A", "A"}, // names are per folder
	}
	for _, tt := range tests {
		if got := tt.folder.UniqueName(tt.base); got != tt.want {
			t.Errorf("UniqueName(%q) = %q, want %q", tt.base, got, tt.want)
		}
	}
	root.Requests = append(root.Requests, &Request{Name: "r1 2"})
	if got := root.UniqueName("r1"); got != "r1 3" {
		t.Errorf("skips taken numbers: %q", got)
	}
}

func TestAddKeepsEarlierPointersValid(t *testing.T) {
	root, _, _, _, _, _, _, _, r1 := fixture()
	added := root.AddRequest(Request{Name: "r1", URL: "u"})
	for i := 0; i < 50; i++ {
		root.AddRequest(Request{Name: "more"})
	}
	if added.Name != "r1 2" || added.URL != "u" {
		t.Fatalf("added = %+v", added)
	}
	if root.Requests[0] != r1 || root.Requests[1] != added {
		t.Fatal("pointers must stay valid however many siblings are added")
	}
	f := root.AddFolder("A")
	if f.Name != "A 2" || root.Folders[len(root.Folders)-1] != f {
		t.Fatalf("folder = %+v", f)
	}
}

func TestParentAndContains(t *testing.T) {
	root, a, a1f, _, a1, _, x, _, _ := fixture()
	for _, tt := range []struct {
		node any
		want *Folder
	}{{a1, a}, {x, a1f}, {a1f, a}, {a, root}} {
		if got, ok := root.Parent(tt.node); !ok || got != tt.want {
			t.Errorf("Parent(%v) = %v %v", tt.node, got, ok)
		}
	}
	if _, ok := root.Parent(root); ok {
		t.Error("the root has no parent")
	}
	if _, ok := root.Parent(&Request{Name: "stranger"}); ok {
		t.Error("a node outside the tree has no parent")
	}
	if !a.Contains(x) || !a.Contains(a) || a.Contains(root) || a1f.Contains(a1) {
		t.Error("Contains is wrong")
	}
}

func TestRename(t *testing.T) {
	root, a, _, _, a1, _, _, _, _ := fixture()
	tests := []struct {
		name string
		node any
		to   string
		want error
		got  string
	}{
		{"plain", a1, "renamed", nil, "renamed"},
		{"trimmed", a1, "  spaced  ", nil, "spaced"},
		{"same name is fine", a1, "a1", nil, "a1"},
		{"empty", a1, "   ", ErrEmptyName, "a1"},
		{"taken by a request", a1, "a2", ErrNameTaken, "a1"},
		{"taken by a folder", a1, "A1", ErrNameTaken, "a1"},
		{"a folder", a, "Alpha", nil, "Alpha"},
		{"a folder onto a sibling", a, "B", ErrNameTaken, "A"},
		{"the root", root, "Collection", nil, "Collection"},
	}
	for _, tt := range tests {
		err := root.Rename(tt.node, tt.to)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
		got := ""
		switch n := tt.node.(type) {
		case *Request:
			got = n.Name
		case *Folder:
			got = n.Name
		}
		if got != tt.got {
			t.Errorf("%s: name = %q, want %q", tt.name, got, tt.got)
		}
		if tt.node == any(a1) {
			a1.Name = "a1"
		}
		if tt.node == any(a) {
			a.Name = "A"
		}
	}
}

func TestDuplicateRequestInsertsACopyAfterTheOriginal(t *testing.T) {
	root, a, _, _, a1, a2, _, _, _ := fixture()
	a1.Params = []KV{{Name: "q", Value: "1"}}
	cp := root.Duplicate(a1).(*Request)
	if got := names(a.Requests); !reflect.DeepEqual(got, []string{"a1", "a1 copy", "a2"}) {
		t.Fatalf("requests = %v", got)
	}
	cp.Params[0].Value = "changed"
	if a1.Params[0].Value != "1" {
		t.Fatal("the copy must not share rows with the original")
	}
	if cp == a1 || a.Requests[0] != a1 || a.Requests[2] != a2 {
		t.Fatal("pointers of the originals must stay")
	}
	again := root.Duplicate(a1).(*Request)
	if again.Name != "a1 copy 2" {
		t.Fatalf("second copy: %q", again.Name)
	}
}

func TestDuplicateFolderCopiesEverythingInIt(t *testing.T) {
	root, _, a1f, _, _, _, x, _, _ := fixture()
	cp := root.Duplicate(root.Folders[0]).(*Folder)
	if got := folderNames(root.Folders); !reflect.DeepEqual(got, []string{"A", "A copy", "B"}) {
		t.Fatalf("folders = %v", got)
	}
	if len(cp.Folders) != 1 || cp.Folders[0] == a1f || cp.Folders[0].Requests[0] == x || cp.Folders[0].Requests[0].Name != "x" {
		t.Fatal("a folder copy is deep")
	}
	if root.Duplicate(root) != nil {
		t.Fatal("the root cannot be duplicated")
	}
}

func TestDelete(t *testing.T) {
	root, a, _, _, a1, _, x, _, _ := fixture()
	if !root.Delete(a1) || !reflect.DeepEqual(names(a.Requests), []string{"a2"}) {
		t.Fatalf("requests = %v", names(a.Requests))
	}
	if !root.Delete(a) || root.Contains(x) || !reflect.DeepEqual(folderNames(root.Folders), []string{"B"}) {
		t.Fatal("deleting a folder removes everything below it")
	}
	if root.Delete(root) {
		t.Fatal("the root cannot be deleted")
	}
	if root.Delete(&Request{Name: "stranger"}) {
		t.Fatal("deleting something that is not in the tree reports false")
	}
}

func TestMove(t *testing.T) {
	t.Run("a request to another folder, at the end", func(t *testing.T) {
		root, a, _, b, a1, _, _, b1, _ := fixture()
		if err := root.Move(a1, b, nil); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(names(b.Requests), []string{"b1", "a1"}) || !reflect.DeepEqual(names(a.Requests), []string{"a2"}) {
			t.Fatalf("A %v B %v", names(a.Requests), names(b.Requests))
		}
		if b.Requests[1] != a1 || b.Requests[0] != b1 {
			t.Fatal("a moved request keeps its identity")
		}
	})
	t.Run("before a sibling", func(t *testing.T) {
		root, _, _, b, a1, _, _, b1, _ := fixture()
		root.Move(a1, b, b1)
		if !reflect.DeepEqual(names(b.Requests), []string{"a1", "b1"}) {
			t.Fatalf("B = %v", names(b.Requests))
		}
	})
	t.Run("reordering inside a folder", func(t *testing.T) {
		root, a, _, _, a1, a2, _, _, _ := fixture()
		if err := root.Move(a2, a, a1); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(names(a.Requests), []string{"a2", "a1"}) {
			t.Fatalf("A = %v", names(a.Requests))
		}
	})
	t.Run("a folder into another", func(t *testing.T) {
		root, a, a1f, b, _, _, _, _, _ := fixture()
		if err := root.Move(b, a, nil); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(folderNames(a.Folders), []string{"A1", "B"}) || !reflect.DeepEqual(folderNames(root.Folders), []string{"A"}) {
			t.Fatalf("A %v root %v", folderNames(a.Folders), folderNames(root.Folders))
		}
		_ = a1f
	})
	t.Run("a folder into itself or below itself", func(t *testing.T) {
		root, a, a1f, _, _, _, _, _, _ := fixture()
		if err := root.Move(a, a, nil); !errors.Is(err, ErrIntoItself) {
			t.Errorf("into itself: %v", err)
		}
		if err := root.Move(a, a1f, nil); !errors.Is(err, ErrIntoItself) {
			t.Errorf("into a descendant: %v", err)
		}
		if !reflect.DeepEqual(folderNames(root.Folders), []string{"A", "B"}) {
			t.Fatal("a refused move must change nothing")
		}
	})
	t.Run("a name clash in the target", func(t *testing.T) {
		root, a, _, b, _, _, _, _, _ := fixture()
		b.Requests = append(b.Requests, &Request{Name: "a1"})
		if err := root.Move(a.Requests[0], b, nil); !errors.Is(err, ErrNameTaken) {
			t.Fatalf("err = %v", err)
		}
		if len(a.Requests) != 2 {
			t.Fatal("a refused move must change nothing")
		}
	})
	t.Run("the root cannot move", func(t *testing.T) {
		root, a, _, _, _, _, _, _, _ := fixture()
		if err := root.Move(root, a, nil); err == nil {
			t.Fatal("want an error")
		}
	})
}
