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
	dir, before := DropPlace(root, model.Node{Req: first}, model.Node{Dir: sub})
	if dir != sub || before != (model.Node{}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestDropPlaceOnARequestLandsBeforeItInItsFolder(t *testing.T) {
	root, first, second, sub := sample()
	dir, before := DropPlace(root, model.Node{Req: first}, model.Node{Req: second})
	if dir != sub || before != (model.Node{Req: second}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestAFolderIsNeverPlacedBeforeARequest(t *testing.T) {
	root, _, second, sub := sample()
	other := root.AddFolder("other")
	dir, before := DropPlace(root, model.Node{Dir: other}, model.Node{Req: second})
	if dir != sub || before != (model.Node{}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestInsideCoversTheRequestItselfAndAFoldersContents(t *testing.T) {
	root, first, second, sub := sample()
	tests := []struct {
		name string
		node model.Node
		req  *model.Request
		want bool
	}{
		{"the request itself", model.Node{Req: first}, first, true},
		{"another request", model.Node{Req: first}, second, false},
		{"a request in the folder", model.Node{Dir: sub}, second, true},
		{"a request outside the folder", model.Node{Dir: sub}, first, false},
		{"the root holds everything", model.Node{Dir: root}, second, true},
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
	if title, body := DeletePrompt(model.Node{Req: first}); title != "Delete first?" || body != "This cannot be undone." {
		t.Fatalf("request: %q %q", title, body)
	}
	if title, body := DeletePrompt(model.Node{Dir: sub}); title != "Delete sub?" || body != "The folder and everything in it will be removed." {
		t.Fatalf("folder: %q %q", title, body)
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
