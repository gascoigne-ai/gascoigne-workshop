package collection

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestTreeRows(t *testing.T) {
	user := &model.Folder{Name: "User", Requests: []*model.Request{{Name: "Me"}}}
	repo := &model.Folder{Name: "Repo", Requests: []*model.Request{{Name: "Search"}, {Name: "Info"}}}
	root := &model.Folder{Name: "API", Folders: []*model.Folder{user, repo}, Requests: []*model.Request{{Name: "Root call"}}}
	tests := []struct {
		name     string
		expanded map[*model.Folder]bool
		want     []string
	}{
		{"collapsed root", nil, []string{"API"}},
		{"root open", map[*model.Folder]bool{root: true}, []string{"API", " User", " Repo", " Root call"}},
		{"one folder open", map[*model.Folder]bool{root: true, repo: true},
			[]string{"API", " User", " Repo", "  Search", "  Info", " Root call"}},
		{"hidden child stays hidden", map[*model.Folder]bool{repo: true}, []string{"API"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, r := range Rows(root, tt.expanded) {
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

func TestTreeRowsCarryTheNodes(t *testing.T) {
	req := &model.Request{Name: "r"}
	sub := &model.Folder{Name: "sub"}
	root := &model.Folder{Name: "A", Folders: []*model.Folder{sub}, Requests: []*model.Request{req}}
	rows := Rows(root, map[*model.Folder]bool{root: true})
	if rows[0].Node() != any(root) || rows[1].Node() != any(sub) || rows[2].Node() != any(req) {
		t.Fatal("each row must stand for its own folder or request, by pointer")
	}
	rows[2].Req.URL = "edited"
	if req.URL != "edited" {
		t.Fatal("a request row aliases the tree's request so edits persist")
	}
}

func TestExpansionSurvivesRenamingAFolder(t *testing.T) {
	sub := &model.Folder{Name: "old", Requests: []*model.Request{{Name: "r"}}}
	root := &model.Folder{Name: "A", Folders: []*model.Folder{sub}}
	open := map[*model.Folder]bool{root: true, sub: true}
	sub.Name = "new"
	if got := len(Rows(root, open)); got != 3 {
		t.Fatalf("rows = %d: a renamed folder must stay expanded", got)
	}
}

func TestSplitMatch(t *testing.T) {
	tests := []struct {
		name, text, filter   string
		before, match, after string
	}{
		{"match in the middle", "List users", "user", "List ", "user", "s"},
		{"case-insensitive keeps the original case", "Get USER", "user", "Get ", "USER", ""},
		{"first occurrence only", "ab ab", "ab", "", "ab", " ab"},
		{"filter is trimmed", "Search", "  arc ", "Se", "arc", "h"},
		{"empty filter", "Search", "", "Search", "", ""},
		{"no match", "Search", "zzz", "Search", "", ""},
		{"multibyte text", "Zażółć gęślą", "gęś", "Zażółć ", "gęś", "lą"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before, match, after := SplitMatch(tc.text, tc.filter)
			if before != tc.before || match != tc.match || after != tc.after {
				t.Errorf("SplitMatch(%q, %q) = %q, %q, %q; want %q, %q, %q",
					tc.text, tc.filter, before, match, after, tc.before, tc.match, tc.after)
			}
		})
	}
}
