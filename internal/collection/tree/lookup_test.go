package tree

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestPathOfJoinsFolderNamesAndEndsWithTheRequest(t *testing.T) {
	root, first, second, _ := sample()
	stranger := &model.Request{Name: "first"}
	tests := []struct {
		name string
		req  *model.Request
		want string
	}{
		{"request in the root", first, "first"},
		{"request in a folder", second, "sub/second"},
		{"request of no tree", stranger, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PathOf(root, tc.req); got != tc.want {
				t.Fatalf("PathOf = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAtReturnsTheFirstOfTwoRequestsWithOnePath(t *testing.T) {
	root := &model.Folder{Name: "root"}
	one := root.AddFolder("dir").AddRequest(model.Request{Name: "dup"})
	root.Folders = append(root.Folders, &model.Folder{Name: "dir", Requests: []*model.Request{{Name: "dup"}}})
	two := root.Folders[1].Requests[0]
	if PathOf(root, one) != "dir/dup" || PathOf(root, two) != "dir/dup" {
		t.Fatal("both requests have the path of their names")
	}
	if At(root, "dir/dup") != one {
		t.Fatal("At must return the first request of that path")
	}
}

func TestAtFindsTheRequestAPathNames(t *testing.T) {
	root, first, second, _ := sample()
	tests := []struct {
		name string
		path string
		want *model.Request
	}{
		{"root request", "first", first},
		{"nested request", "sub/second", second},
		{"a folder is not a request", "sub", nil},
		{"unknown path", "sub/third", nil},
		{"empty path", "", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := At(root, tc.path); got != tc.want {
				t.Fatalf("At(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestWalkRequestsVisitsRootRequestsBeforeFolders(t *testing.T) {
	root, _, _, _ := sample()
	var paths []string
	WalkRequests(root, "", func(path string, _ *model.Request) { paths = append(paths, path) })
	if len(paths) != 2 || paths[0] != "first" || paths[1] != "sub/second" {
		t.Fatalf("paths %v", paths)
	}
}

func TestWalkFoldersVisitsNestedFoldersWithTheirPath(t *testing.T) {
	root, _, _, sub := sample()
	sub.AddFolder("deep")
	var paths []string
	WalkFolders(root, "", func(path string, _ *model.Folder) { paths = append(paths, path) })
	if len(paths) != 2 || paths[0] != "sub" || paths[1] != "sub/deep" {
		t.Fatalf("paths %v", paths)
	}
}
