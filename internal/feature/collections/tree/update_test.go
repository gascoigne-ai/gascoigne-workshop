package tree

import (
	"errors"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func sampleCollection() (col *request.Collection, first, second *request.Request, sub *request.Folder) {
	root, first, second, sub := sample()
	return &request.Collection{Folder: *root}, first, second, sub
}

func TestRenameRefusedNameIsReportedAndKeepsTheOldOne(t *testing.T) {
	col, first, _, _ := sampleCollection()
	got := Update(col, request.Node{}, Rename{Node: request.Node{Req: first}, Name: "  "})
	if !errors.Is(got.Err, request.ErrEmptyName) || first.Name != "first" {
		t.Fatalf("err %v name %q", got.Err, first.Name)
	}
	got = Update(col, request.Node{}, Rename{Node: request.Node{Req: first}, Name: "renamed"})
	if got.Err != nil || first.Name != "renamed" {
		t.Fatalf("err %v name %q", got.Err, first.Name)
	}
}

func TestDuplicateSelectsTheCopy(t *testing.T) {
	col, first, _, _ := sampleCollection()
	got := Update(col, request.Node{}, Duplicate{Node: request.Node{Req: first}})
	if got.Selected.Req == nil || got.Selected.Req == first || got.Selected.Req.Name != "first copy" {
		t.Fatalf("selected %+v", got.Selected)
	}
}

func TestDeleteRemovesTheNodeAndClearsItsSelection(t *testing.T) {
	col, first, _, sub := sampleCollection()
	got := Update(col, request.Node{Req: first}, Delete{Node: request.Node{Req: first}})
	if got.Selected != (request.Node{}) || got.Removed.Req != first || len(col.Folder.Requests) != 0 {
		t.Fatalf("%+v requests %d", got, len(col.Folder.Requests))
	}
	got = Update(col, request.Node{Req: first}, Delete{Node: request.Node{Dir: sub}})
	if got.Removed.Dir != sub || len(col.Folder.Folders) != 0 {
		t.Fatalf("folder not removed: %+v", got)
	}
}

func TestDeleteLeavesAnotherSelectionAndRefusesTheRoot(t *testing.T) {
	col, first, second, _ := sampleCollection()
	got := Update(col, request.Node{Req: first}, Delete{Node: request.Node{Req: second}})
	if got.Selected.Req != first {
		t.Fatalf("selection changed: %+v", got.Selected)
	}
	got = Update(col, request.Node{}, Delete{Node: request.Node{Dir: &col.Folder}})
	if got.Removed != (request.Node{}) {
		t.Fatal("the root cannot be deleted")
	}
	if got = Update(col, request.Node{}, Delete{}); got.Removed != (request.Node{}) {
		t.Fatal("nothing to delete")
	}
}

func TestNewRequestAndNewFolderAddToTheFolder(t *testing.T) {
	col, _, _, sub := sampleCollection()
	got := Update(col, request.Node{}, NewRequest{Dir: sub})
	if got.Open == nil || got.Open.Name != "New Request" || got.Open.Method != "GET" || got.Expand != sub {
		t.Fatalf("%+v", got)
	}
	got = Update(col, request.Node{}, NewFolder{Dir: sub})
	if got.Selected.Dir == nil || got.Selected.Dir.Name != "New Folder" || got.Expand != sub {
		t.Fatalf("%+v", got)
	}
}
