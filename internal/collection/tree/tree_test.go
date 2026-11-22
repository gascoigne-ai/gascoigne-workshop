package tree

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func sample() (root *request.Folder, first, second *request.Request, sub *request.Folder) {
	root = &request.Folder{Name: "root"}
	first = root.AddRequest(request.Request{Name: "first"})
	sub = root.AddFolder("sub")
	second = sub.AddRequest(request.Request{Name: "second"})
	return root, first, second, sub
}

func TestDropPlaceIsTheFolderForAFolderRow(t *testing.T) {
	root, first, _, sub := sample()
	dir, before := DropPlace(root, request.Node{Req: first}, request.Node{Dir: sub})
	if dir != sub || before != (request.Node{}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestDropPlaceOnARequestLandsBeforeItInItsFolder(t *testing.T) {
	root, first, second, sub := sample()
	dir, before := DropPlace(root, request.Node{Req: first}, request.Node{Req: second})
	if dir != sub || before != (request.Node{Req: second}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestAFolderIsNeverPlacedBeforeARequest(t *testing.T) {
	root, _, second, sub := sample()
	other := root.AddFolder("other")
	dir, before := DropPlace(root, request.Node{Dir: other}, request.Node{Req: second})
	if dir != sub || before != (request.Node{}) {
		t.Fatalf("dir %v before %v", dir, before)
	}
}

func TestInsideCoversTheRequestItselfAndAFoldersContents(t *testing.T) {
	root, first, second, sub := sample()
	tests := []struct {
		name string
		node request.Node
		req  *request.Request
		want bool
	}{
		{"the request itself", request.Node{Req: first}, first, true},
		{"another request", request.Node{Req: first}, second, false},
		{"a request in the folder", request.Node{Dir: sub}, second, true},
		{"a request outside the folder", request.Node{Dir: sub}, first, false},
		{"the root holds everything", request.Node{Dir: root}, second, true},
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
	if title, body := DeletePrompt(request.Node{Req: first}); title != "Delete first?" || body != "This cannot be undone." {
		t.Fatalf("request: %q %q", title, body)
	}
	if title, body := DeletePrompt(request.Node{Dir: sub}); title != "Delete sub?" || body != "The folder and everything in it will be removed." {
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
	if FirstRequest(&request.Folder{}) != nil {
		t.Fatal("an empty folder has none")
	}
	nested := &request.Folder{Folders: []*request.Folder{{}, sub}}
	if FirstRequest(nested) != second {
		t.Fatal("empty subfolders are skipped")
	}
}

func TestKeyChangesWithTheCollectionAndNotWithoutIt(t *testing.T) {
	root, first, _, _ := sample()
	collection := request.Collection{Folder: *root}
	before := Key(collection)
	if Key(collection) != before {
		t.Fatal("the key of the same collection is stable")
	}
	first.URL = "http://changed"
	if Key(collection) == before {
		t.Fatal("an edit changes the key")
	}
}
