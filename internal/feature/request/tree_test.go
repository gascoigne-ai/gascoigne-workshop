package request

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
		node Node
		want *Folder
	}{{Of(a1), a}, {Of(x), a1f}, {Of(a1f), a}, {Of(a), root}} {
		if got, ok := root.Parent(tt.node); !ok || got != tt.want {
			t.Errorf("Parent(%v) = %v %v", tt.node, got, ok)
		}
	}
	if _, ok := root.Parent(Of(root)); ok {
		t.Error("the root has no parent")
	}
	if _, ok := root.Parent(Of(&Request{Name: "stranger"})); ok {
		t.Error("a node outside the tree has no parent")
	}
}

func TestContainsIsTrueForTheFolderItselfAndAnythingBelow(t *testing.T) {
	root, a, a1f, _, a1, _, x, _, _ := fixture()
	cases := []struct {
		name   string
		folder *Folder
		node   Node
		want   bool
	}{
		{"a request below", a, Of(x), true},
		{"a request directly in it", a, Of(a1), true},
		{"the folder itself", a, Of(a), true},
		{"a subfolder", a, Of(a1f), true},
		{"its parent", a, Of(root), false},
		{"a sibling's request", a1f, Of(a1), false},
		{"a node of no tree", a, Of(&Request{Name: "stranger"}), false},
		{"none", a, Node{}, false},
		{"a typed nil folder", a, Of((*Folder)(nil)), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.folder.Contains(tc.node); got != tc.want {
				t.Errorf("Contains = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRename(t *testing.T) {
	root, a, _, _, a1, _, _, _, _ := fixture()
	tests := []struct {
		name string
		node Node
		to   string
		want error
		got  string
	}{
		{"plain", Of(a1), "renamed", nil, "renamed"},
		{"trimmed", Of(a1), "  spaced  ", nil, "spaced"},
		{"same name is fine", Of(a1), "a1", nil, "a1"},
		{"empty", Of(a1), "   ", ErrEmptyName, "a1"},
		{"taken by a request", Of(a1), "a2", ErrNameTaken, "a1"},
		{"taken by a folder", Of(a1), "A1", ErrNameTaken, "a1"},
		{"a folder", Of(a), "Alpha", nil, "Alpha"},
		{"a folder onto a sibling", Of(a), "B", ErrNameTaken, "A"},
		{"the root", Of(root), "Collection", nil, "Collection"},
	}
	for _, tt := range tests {
		err := root.Rename(tt.node, tt.to)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
		got := ""
		switch {
		case tt.node.Req != nil:
			got = tt.node.Req.Name
		case tt.node.Dir != nil:
			got = tt.node.Dir.Name
		}
		if got != tt.got {
			t.Errorf("%s: name = %q, want %q", tt.name, got, tt.got)
		}
		if tt.node == Of(a1) {
			a1.Name = "a1"
		}
		if tt.node == Of(a) {
			a.Name = "A"
		}
	}
}

func TestDuplicateRequestInsertsACopyAfterTheOriginal(t *testing.T) {
	root, a, _, _, a1, a2, _, _, _ := fixture()
	a1.Params = []KV{{Name: "q", Value: "1"}}
	cp := root.Duplicate(Of(a1)).Req
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
	again := root.Duplicate(Of(a1)).Req
	if again.Name != "a1 copy 2" {
		t.Fatalf("second copy: %q", again.Name)
	}
}

func TestDuplicateFolderCopiesEverythingInIt(t *testing.T) {
	root, _, a1f, _, _, _, x, _, _ := fixture()
	cp := root.Duplicate(Of(root.Folders[0])).Dir
	if got := folderNames(root.Folders); !reflect.DeepEqual(got, []string{"A", "A copy", "B"}) {
		t.Fatalf("folders = %v", got)
	}
	if len(cp.Folders) != 1 || cp.Folders[0] == a1f || cp.Folders[0].Requests[0] == x || cp.Folders[0].Requests[0].Name != "x" {
		t.Fatal("a folder copy is deep")
	}
	if !root.Duplicate(Of(root)).None() {
		t.Fatal("the root cannot be duplicated")
	}
}

func TestDelete(t *testing.T) {
	root, a, _, _, a1, _, x, _, _ := fixture()
	if !root.Delete(Of(a1)) || !reflect.DeepEqual(names(a.Requests), []string{"a2"}) {
		t.Fatalf("requests = %v", names(a.Requests))
	}
	if !root.Delete(Of(a)) || root.Contains(Of(x)) || !reflect.DeepEqual(folderNames(root.Folders), []string{"B"}) {
		t.Fatal("deleting a folder removes everything below it")
	}
	if root.Delete(Of(root)) {
		t.Fatal("the root cannot be deleted")
	}
	if root.Delete(Of(&Request{Name: "stranger"})) {
		t.Fatal("deleting something that is not in the tree reports false")
	}
}

func TestZeroNodeIsNotFoundByEveryTreeOperation(t *testing.T) {
	root, _, _, _, _, _, _, _, _ := fixture()
	if parent, ok := root.Parent(Node{}); ok || parent != nil {
		t.Errorf("Parent of the zero Node = %v %v", parent, ok)
	}
	if err := root.Rename(Node{}, "x"); !errors.Is(err, ErrNotInTree) {
		t.Errorf("Rename of the zero Node: err = %v, want ErrNotInTree", err)
	}
	if got := root.Duplicate(Node{}); !got.None() {
		t.Errorf("Duplicate of the zero Node = %+v", got)
	}
	if root.Delete(Node{}) {
		t.Error("Delete of the zero Node reports true")
	}
}

func TestATypedNilThroughOfIsTheZeroNode(t *testing.T) {
	root, _, _, _, _, _, _, _, _ := fixture()
	for name, node := range map[string]Node{"request": Of((*Request)(nil)), "folder": Of((*Folder)(nil))} {
		if node != (Node{}) {
			t.Errorf("%s: Of(typed nil) = %+v, want the zero Node", name, node)
		}
		if _, ok := root.Parent(node); ok {
			t.Errorf("%s: a typed nil has a parent", name)
		}
	}
}

func TestDuplicateReturnsTheCopyAsANode(t *testing.T) {
	root, a, _, _, a1, _, _, _, _ := fixture()
	request := root.Duplicate(Of(a1))
	if request.Req == nil || request.Dir != nil || request.Req.Name != "a1 copy" {
		t.Errorf("request copy = %+v", request)
	}
	folder := root.Duplicate(Of(a))
	if folder.Dir == nil || folder.Req != nil || folder.Dir.Name != "A copy" {
		t.Errorf("folder copy = %+v", folder)
	}
}
