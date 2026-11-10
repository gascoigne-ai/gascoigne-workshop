package tree

import (
	"errors"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func sampleCollection() (col *model.Collection, first, second *model.Request, sub *model.Folder) {
	root, first, second, sub := sample()
	return &model.Collection{Folder: *root}, first, second, sub
}

func TestRenameRefusedNameIsReportedAndKeepsTheOldOne(t *testing.T) {
	col, first, _, _ := sampleCollection()
	got := Update(col, Node{}, Rename{Node: Node{Req: first}, Name: "  "})
	if !errors.Is(got.Err, model.ErrEmptyName) || first.Name != "first" {
		t.Fatalf("err %v name %q", got.Err, first.Name)
	}
	got = Update(col, Node{}, Rename{Node: Node{Req: first}, Name: "renamed"})
	if got.Err != nil || first.Name != "renamed" {
		t.Fatalf("err %v name %q", got.Err, first.Name)
	}
}

func TestDuplicateSelectsTheCopy(t *testing.T) {
	col, first, _, _ := sampleCollection()
	got := Update(col, Node{}, Duplicate{Node: Node{Req: first}})
	if got.Selected.Req == nil || got.Selected.Req == first || got.Selected.Req.Name != "first copy" {
		t.Fatalf("selected %+v", got.Selected)
	}
}

func TestDeleteRemovesTheNodeAndClearsItsSelection(t *testing.T) {
	col, first, _, sub := sampleCollection()
	got := Update(col, Node{Req: first}, Delete{Node: Node{Req: first}})
	if got.Selected != (Node{}) || got.Removed.Req != first || len(col.Folder.Requests) != 0 {
		t.Fatalf("%+v requests %d", got, len(col.Folder.Requests))
	}
	got = Update(col, Node{Req: first}, Delete{Node: Node{Dir: sub}})
	if got.Removed.Dir != sub || len(col.Folder.Folders) != 0 {
		t.Fatalf("folder not removed: %+v", got)
	}
}

func TestDeleteLeavesAnotherSelectionAndRefusesTheRoot(t *testing.T) {
	col, first, second, _ := sampleCollection()
	got := Update(col, Node{Req: first}, Delete{Node: Node{Req: second}})
	if got.Selected.Req != first {
		t.Fatalf("selection changed: %+v", got.Selected)
	}
	got = Update(col, Node{}, Delete{Node: Node{Dir: &col.Folder}})
	if got.Removed != (Node{}) {
		t.Fatal("the root cannot be deleted")
	}
	if got = Update(col, Node{}, Delete{}); got.Removed != (Node{}) {
		t.Fatal("nothing to delete")
	}
}

func TestMoveExpandsTheTargetFolderAndReportsARefusal(t *testing.T) {
	col, first, second, sub := sampleCollection()
	got := Update(col, Node{}, Move{Node: Node{Req: first}, Target: Node{Dir: sub}})
	if got.Err != nil || got.Expand != sub || len(sub.Requests) != 2 {
		t.Fatalf("%+v", got)
	}
	got = Update(col, Node{}, Move{Node: Node{Dir: sub}, Target: Node{Req: second}})
	if !errors.Is(got.Err, model.ErrIntoItself) {
		t.Fatalf("err %v", got.Err)
	}
}

func TestMoveOntoItselfOrNothingDoesNothing(t *testing.T) {
	col, first, _, _ := sampleCollection()
	for _, msg := range []Move{{Node: Node{Req: first}, Target: Node{Req: first}}, {Node: Node{Req: first}}, {}} {
		if got := Update(col, Node{}, msg); got != (Result{}) {
			t.Fatalf("%+v: %+v", msg, got)
		}
	}
}

func TestNewRequestAndNewFolderAddToTheFolder(t *testing.T) {
	col, _, _, sub := sampleCollection()
	got := Update(col, Node{}, NewRequest{Dir: sub})
	if got.Open == nil || got.Open.Name != "New Request" || got.Open.Method != "GET" || got.Expand != sub {
		t.Fatalf("%+v", got)
	}
	got = Update(col, Node{}, NewFolder{Dir: sub})
	if got.Selected.Dir == nil || got.Selected.Dir.Name != "New Folder" || got.Expand != sub {
		t.Fatalf("%+v", got)
	}
}
