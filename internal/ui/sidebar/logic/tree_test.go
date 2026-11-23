package logic

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// fixture is a root with a folder "api" (holding "ping") and a request "home".
type fixture struct {
	root, api *request.Folder
	ping      *request.Request
	home      *request.Request
}

func newFixture() fixture {
	ping := &request.Request{Name: "ping"}
	home := &request.Request{Name: "home"}
	api := &request.Folder{Name: "api", Requests: []*request.Request{ping}}
	root := &request.Folder{Name: "root", Folders: []*request.Folder{api}, Requests: []*request.Request{home}}
	return fixture{root: root, api: api, ping: ping, home: home}
}

func (fix fixture) rows() []collection.Row {
	return collection.Rows(fix.root, map[*request.Folder]bool{fix.root: true, fix.api: true})
}

func TestStepMovesTheSelectionAndClampsAtTheEnds(t *testing.T) {
	fix := newFixture()
	rows := fix.rows() // root, api, ping, home
	tests := []struct {
		name  string
		start request.Node
		rows  []collection.Row
		by    int
		want  request.Node
	}{
		{"down from nothing selects the first", request.Node{}, rows, 1, rows[0].Node()},
		{"up from nothing selects the first", request.Node{}, rows, -1, rows[0].Node()},
		{"down moves one", rows[1].Node(), rows, 1, rows[2].Node()},
		{"up moves one", rows[2].Node(), rows, -1, rows[1].Node()},
		{"down stops at the last", rows[3].Node(), rows, 1, rows[3].Node()},
		{"up stops at the first", rows[0].Node(), rows, -1, rows[0].Node()},
		{"empty rows change nothing", rows[1].Node(), nil, 1, rows[1].Node()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := New()
			tree.Select(test.start)
			tree.Step(test.rows, test.by)
			if got := tree.Selected(); got != test.want {
				t.Errorf("selected %v, want %v", got, test.want)
			}
		})
	}
}

func TestToggleOpensAndClosesAFolder(t *testing.T) {
	fix := newFixture()
	tree := New()
	tree.Toggle(fix.api)
	if !tree.Expanded()[fix.api] {
		t.Fatal("first toggle left the folder closed")
	}
	tree.Toggle(fix.api)
	if tree.Expanded()[fix.api] {
		t.Fatal("second toggle left the folder open")
	}
}

func TestExpandAllOpensEverySubfolder(t *testing.T) {
	deep := &request.Folder{Name: "deep"}
	mid := &request.Folder{Name: "mid", Folders: []*request.Folder{deep}}
	root := &request.Folder{Name: "root", Folders: []*request.Folder{mid}}
	tree := New()
	tree.ExpandAll(root)
	for _, folder := range []*request.Folder{root, mid, deep} {
		if !tree.Expanded()[folder] {
			t.Errorf("folder %s is closed", folder.Name)
		}
	}
}

func TestResetForgetsSelectionAndExpansion(t *testing.T) {
	fix := newFixture()
	tree := New()
	tree.Select(request.Node{Req: fix.home})
	tree.Expand(fix.api)
	tree.Reset()
	if !tree.Selected().None() || len(tree.Expanded()) != 0 {
		t.Errorf("selected %v, expanded %v after Reset", tree.Selected(), tree.Expanded())
	}
}

func TestRevealSelectsTheRequestAndOpensTheFoldersAbove(t *testing.T) {
	fix := newFixture()
	tree := New()
	tree.Reveal(fix.root, fix.ping)
	if tree.Selected() != (request.Node{Req: fix.ping}) {
		t.Errorf("selected %v", tree.Selected())
	}
	if !tree.Expanded()[fix.root] || !tree.Expanded()[fix.api] {
		t.Errorf("root and api must be open, got %v", tree.Expanded())
	}
}

func TestFocusAndBlurMoveTheKeyboard(t *testing.T) {
	tree := New()
	tree.Focus()
	if !tree.Focused() {
		t.Fatal("Focus did not focus")
	}
	tree.Blur()
	if tree.Focused() {
		t.Fatal("Blur did not blur")
	}
}
