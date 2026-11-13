package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// sidebarPinH is the height the sidebar is pinned at: room for every row
// of the sample collection.
const sidebarPinH = 480

// The sidebar is pinned as text (element kinds, options, rectangles) at
// its default and its smallest width in its key states: the tree as it
// opens (a selected request, open and closed folders), a selected
// folder, a filter that matches
// names and one that matches only URLs, a drag over another row and over
// itself, and the log of sent requests.
func TestTheSidebarLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name  string
		setup func(a *App)
	}{
		{"tree", func(a *App) {}},
		{"folder-selected", func(a *App) { click(a, 100, treeRowY(rowRealtime)) }},
		{"filter-name", func(a *App) {
			click(a, 180, headerH/2)
			a.typeText("tags")
		}},
		{"filter-url", func(a *App) {
			click(a, 180, headerH/2)
			a.typeText("search/repos")
		}},
		{"drop", func(a *App) {
			a.dragNode = findReq(a, "Repository Info")
			a.dropTarget = a.col.Folders[1]
		}},
		{"drop-self", func(a *App) {
			info := findReq(a, "Repository Info")
			a.dragNode, a.dropTarget = info, info
		}},
		{"sent", func(a *App) { a.showSent = true }},
	}
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", sidebarW - 1},
		{"narrow", minSidebarW},
	}
	for _, tc := range cases {
		for _, size := range sizes {
			name := "sidebar-" + tc.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				tc.setup(a)
				root := a.sidebar()
				Layout(a.ctx, root, size.width, sidebarPinH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// The chip in the sidebar's header switches between the tree and the log
// of sent requests, and its label names what is shown.
func TestTheSidebarChipSwitchesBetweenTreeAndSentLog(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	clickText(t, a, "Collections", 0)
	if !a.showSent {
		t.Fatal("clicking Collections did not show the sent log")
	}
	clickText(t, a, "Sent", 0)
	if a.showSent {
		t.Fatal("clicking Sent did not show the tree")
	}
	textCenter(t, a, "Collections")
}

// While a row is dragged, the row under the pointer, folder or request,
// is where it would land.
func TestDraggingOverATreeRowMakesItTheDropTarget(t *testing.T) {
	cases := []struct {
		name   string
		row    int
		target func(a *App) any
	}{
		{"request", rowSearch, func(a *App) any { return findReq(a, "Search Repos") }},
		{"folder", rowUser, func(a *App) any { return a.col.Folders[0] }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			a.step(core.Input{X: 100, Y: treeRowY(rowInfo), Click: true, Held: true})
			a.step(core.Input{X: 100, Y: treeRowY(tc.row), Held: true})
			if a.dragNode != findReq(a, "Repository Info") {
				t.Fatalf("dragging %v, want Repository Info", a.dragNode)
			}
			if want := tc.target(a); a.dropTarget != want {
				t.Fatalf("drop target %v, want %v", a.dropTarget, want)
			}
		})
	}
}
