package tree

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func sample() (root *model.Folder, first, second *model.Request, sub *model.Folder) {
	root = &model.Folder{Name: "root"}
	first = root.AddRequest(model.Request{Name: "first"})
	sub = root.AddFolder("sub")
	second = sub.AddRequest(model.Request{Name: "second"})
	return root, first, second, sub
}

func TestDropPlaceIsTheFolderForAFolderRow(t *testing.T) {
	root, first, _, sub := sample()
	dir, before := DropPlace(root, Node{Req: first}, Node{Dir: sub})
	if dir != sub || before != (Node{}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestDropPlaceOnARequestLandsBeforeItInItsFolder(t *testing.T) {
	root, first, second, sub := sample()
	dir, before := DropPlace(root, Node{Req: first}, Node{Req: second})
	if dir != sub || before != (Node{Req: second}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestAFolderIsNeverPlacedBeforeARequest(t *testing.T) {
	root, _, second, sub := sample()
	other := root.AddFolder("other")
	dir, before := DropPlace(root, Node{Dir: other}, Node{Req: second})
	if dir != sub || before != (Node{}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestInsideCoversTheRequestItselfAndAFoldersContents(t *testing.T) {
	root, first, second, sub := sample()
	tests := []struct {
		name string
		node Node
		req  *model.Request
		want bool
	}{
		{"the request itself", Node{Req: first}, first, true},
		{"another request", Node{Req: first}, second, false},
		{"a request in the folder", Node{Dir: sub}, second, true},
		{"a request outside the folder", Node{Dir: sub}, first, false},
		{"the root holds everything", Node{Dir: root}, second, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Inside(test.node, test.req); got != test.want {
				t.Fatalf("Inside = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDeletePromptWarnsMoreForAFolder(t *testing.T) {
	_, first, _, sub := sample()
	if title, body := DeletePrompt(Node{Req: first}); title != "Delete first?" || body != "This cannot be undone." {
		t.Fatalf("request: %q %q", title, body)
	}
	if title, body := DeletePrompt(Node{Dir: sub}); title != "Delete sub?" || body != "The folder and everything in it will be removed." {
		t.Fatalf("folder: %q %q", title, body)
	}
}

func TestNodeNameIsEmptyForAnUnknownNode(t *testing.T) {
	_, first, _, sub := sample()
	if Of(first).Name() != "first" || Of(sub).Name() != "sub" || Of(42).Name() != "" {
		t.Fatal("Node must read requests and folders only")
	}
}

func TestFirstRequestIsTheFirstInTreeOrder(t *testing.T) {
	root, first, second, sub := sample()
	if FirstRequest(root) != first {
		t.Fatal("a request of the folder itself comes before its subfolders")
	}
	if FirstRequest(sub) != second {
		t.Fatal("a subfolder's own first request")
	}
	if FirstRequest(&model.Folder{}) != nil {
		t.Fatal("an empty folder has none")
	}
	nested := &model.Folder{Folders: []*model.Folder{{}, sub}}
	if FirstRequest(nested) != second {
		t.Fatal("empty subfolders are skipped")
	}
}

func TestKeyChangesWithTheCollectionAndNotWithoutIt(t *testing.T) {
	root, first, _, _ := sample()
	collection := model.Collection{Folder: *root}
	before := Key(collection)
	if Key(collection) != before {
		t.Fatal("the key of the same collection is stable")
	}
	first.URL = "http://changed"
	if Key(collection) == before {
		t.Fatal("an edit changes the key")
	}
}
