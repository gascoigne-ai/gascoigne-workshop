package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// varsFixture is a collection with a variable of its own, an environment
// with one and a folder with one.
func varsFixture() (col, other *request.Collection, folder *request.Folder) {
	folder = &request.Folder{Name: "Users", Vars: []request.KV{{Name: "folder"}}}
	col = &request.Collection{Folder: request.Folder{Name: "Alpha", Folders: []*request.Folder{folder}}, Vars: []request.KV{{Name: "col"}},
		Envs: []request.Environment{{Name: "Dev", Vars: []request.KV{{Name: "env"}}}}}
	other = &request.Collection{Folder: request.Folder{Name: "Beta"}}
	return col, other, folder
}

func TestAVarsTargetSaysWhatItEditsAndNamesIt(t *testing.T) {
	col, _, folder := varsFixture()
	cases := []struct {
		name        string
		target      varsTarget
		env         int // the entry's active environment, -1 for none
		wantRows    *[]request.KV
		wantCaption string
	}{
		{"active with an environment", varsTarget{}, 0, &col.Envs[0].Vars, "Environment: Dev"},
		{"active without one", varsTarget{}, -1, &col.Vars, "Collection variables"},
		{"a folder with an environment", folderVars(folder), 0, &folder.Vars, "Folder: Users"},
		{"a folder without one", folderVars(folder), -1, &folder.Vars, "Folder: Users"},
		{"the collection with an environment", collectionVars(col), 0, &col.Vars, "Collection variables"},
		{"the collection without one", collectionVars(col), -1, &col.Vars, "Collection variables"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := collections.Entry{Collection: col, Env: tc.env}
			if got := tc.target.rows(entry); got != tc.wantRows {
				t.Errorf("rows = %p, want %p", got, tc.wantRows)
			}
			if got := tc.target.caption(entry.Environment()); got != tc.wantCaption {
				t.Errorf("caption = %q, want %q", got, tc.wantCaption)
			}
		})
	}
}

func TestAnActiveVarsTargetWithNothingOpenEditsAScratchList(t *testing.T) {
	rows := varsTarget{}.rows(collections.Entry{})
	if rows == nil {
		t.Fatal("no rows to edit")
	}
	*rows = append(*rows, request.KV{Name: "x"})
	if again := (varsTarget{}).rows(collections.Entry{}); len(*again) != 0 {
		t.Error("the scratch list is kept")
	}
}

func TestAVarsTargetRefersToTheCollectionItsVariablesBelongTo(t *testing.T) {
	col, other, folder := varsFixture()
	cases := []struct {
		name   string
		target varsTarget
		col    *request.Collection
		want   bool
	}{
		{"active", varsTarget{}, col, false},
		{"a folder of the collection", folderVars(folder), col, true},
		{"a folder of another", folderVars(folder), other, false},
		{"the collection", collectionVars(col), col, true},
		{"another collection", collectionVars(col), other, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.target.refersTo(tc.col); got != tc.want {
				t.Errorf("refersTo = %v, want %v", got, tc.want)
			}
		})
	}
}
