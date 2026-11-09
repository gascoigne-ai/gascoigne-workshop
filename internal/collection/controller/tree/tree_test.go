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
	dir, before := DropPlace(root, first, sub)
	if dir != sub || before != nil {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestDropPlaceOnARequestLandsBeforeItInItsFolder(t *testing.T) {
	root, first, second, sub := sample()
	dir, before := DropPlace(root, first, second)
	if dir != sub || before != any(second) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestAFolderIsNeverPlacedBeforeARequest(t *testing.T) {
	root, _, second, sub := sample()
	other := root.AddFolder("other")
	dir, before := DropPlace(root, other, second)
	if dir != sub || before != nil {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestInsideCoversTheRequestItselfAndAFoldersContents(t *testing.T) {
	root, first, second, sub := sample()
	tests := []struct {
		name string
		node any
		req  *model.Request
		want bool
	}{
		{"the request itself", first, first, true},
		{"another request", first, second, false},
		{"a request in the folder", sub, second, true},
		{"a request outside the folder", sub, first, false},
		{"the root holds everything", root, second, true},
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
	if title, body := DeletePrompt(first); title != "Delete first?" || body != "This cannot be undone." {
		t.Fatalf("request: %q %q", title, body)
	}
	if title, body := DeletePrompt(sub); title != "Delete sub?" || body != "The folder and everything in it will be removed." {
		t.Fatalf("folder: %q %q", title, body)
	}
}

func TestNameOfIsEmptyForAnUnknownNode(t *testing.T) {
	_, first, _, sub := sample()
	if NameOf(first) != "first" || NameOf(sub) != "sub" || NameOf(42) != "" {
		t.Fatal("NameOf must read requests and folders only")
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
