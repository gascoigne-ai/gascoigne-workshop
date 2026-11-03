package ui

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestTreeRows(t *testing.T) {
	root := model.Folder{
		Name: "API",
		Folders: []model.Folder{
			{Name: "User", Requests: []model.Request{{Name: "Me"}}},
			{Name: "Repo", Requests: []model.Request{{Name: "Search"}, {Name: "Info"}}},
		},
		Requests: []model.Request{{Name: "Root call"}},
	}
	tests := []struct {
		name     string
		expanded map[string]bool
		want     []string
	}{
		{"collapsed root", nil, []string{"API"}},
		{"root open", map[string]bool{"API": true}, []string{"API", " User", " Repo", " Root call"}},
		{"one folder open", map[string]bool{"API": true, "API/Repo": true},
			[]string{"API", " User", " Repo", "  Search", "  Info", " Root call"}},
		{"hidden child stays hidden", map[string]bool{"API/Repo": true}, []string{"API"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, r := range treeRows(root, tt.expanded) {
				pad := ""
				for range r.Depth {
					pad += " "
				}
				got = append(got, pad+r.Name)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTreeRowsRequestPointsIntoTree(t *testing.T) {
	root := model.Folder{Name: "A", Requests: []model.Request{{Name: "r"}}}
	rows := treeRows(root, map[string]bool{"A": true})
	rows[1].Req.URL = "edited"
	if root.Requests[0].URL != "edited" {
		t.Fatal("request row must alias the tree's request so edits persist")
	}
}
