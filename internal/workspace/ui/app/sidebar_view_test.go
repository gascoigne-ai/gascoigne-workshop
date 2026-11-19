package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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
// itself, the log of sent requests, and the prompt left when no collection
// is open.
func TestTheSidebarLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name  string
		setup func(app *App)
	}{
		{"tree", func(app *App) {}},
		{"folder-selected", func(app *App) { click(app, 100, treeRowY(rowRealtime)) }},
		{"filter-name", func(app *App) {
			click(app, 180, headerH/2)
			app.typeText("tags")
		}},
		{"filter-url", func(app *App) {
			click(app, 180, headerH/2)
			app.typeText("search/repos")
		}},
		{"drop", func(app *App) {
			app.dragNode = model.Of(findReq(app, "Repository Info"))
			app.dropTarget = model.Of(app.col().Folders[1])
		}},
		{"drop-self", func(app *App) {
			info := findReq(app, "Repository Info")
			app.dragNode, app.dropTarget = model.Of(info), model.Of(info)
		}},
		{"sent", func(app *App) { app.showSent = true }},
		{"empty", func(app *App) { app.closeCollection(app.col()) }},
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

// The part of a folder or request name the filter matched is drawn in the
// accent meant for lines and marks on a plain background. It was drawn in
// the text color meant for an accent fill, which on the sunken sidebar is
// nearly the background color, so the match was invisible.
func TestTheFilterMatchInATreeRowIsDrawnInTheAccentLine(t *testing.T) {
	cases := []struct {
		name   string
		filter string
		match  string
	}{
		{"folder", "user", "User"},
		{"request", "tags", "Tags"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			click(a, 180, headerH/2)
			a.typeText(tc.filter)
			root := a.sidebar()
			Layout(a.ctx, root, sidebarW-1, sidebarPinH)
			line := dumpLineWithText(t, Dump(root), tc.match)
			if want := "fg=" + string(core.AccentLine); !strings.Contains(line, want) {
				t.Fatalf("matched part drawn as %q, want %s", line, want)
			}
		})
	}
}

// With several roots in one tree, a collection's name must stand out from
// its folders: the root row's name is bold, with or without the filter
// splitting it, and a folder's is not.
func TestARootRowNameIsBoldAndAFolderNameIsNot(t *testing.T) {
	cases := []struct {
		name   string
		filter string
		text   string
		bold   bool
	}{
		{"root", "", "Alpha", true},
		{"second root", "", "Beta", true},
		{"folder", "", "Nested", false},
		{"root matched by the filter", "Alph", "Alph", true},
		{"root after the filtered part", "Alph", "a", true},
		{"folder matched by the filter", "Nest", "Nest", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _, _ := twoCollectionsApp(t)
			t.Cleanup(a.waitWrites)
			a.filter.Text = tc.filter
			root := a.sidebar()
			Layout(a.ctx, root, sidebarW-1, sidebarPinH)
			line := dumpLineWithText(t, Dump(root), tc.text)
			if got := strings.Contains(line, "bold=true"); got != tc.bold {
				t.Fatalf("bold %v, want %v: %s", got, tc.bold, line)
			}
		})
	}
}

// dumpLineWithText is the first line of a dump whose element shows text
// exactly.
func dumpLineWithText(t *testing.T, dump, text string) string {
	t.Helper()
	needle := fmt.Sprintf("text=%q", text)
	for _, line := range strings.Split(dump, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	t.Fatalf("no element with text %q in:\n%s", text, dump)
	return ""
}

// While a row is dragged, the row under the pointer, folder or request,
// is where it would land.
func TestDraggingOverATreeRowMakesItTheDropTarget(t *testing.T) {
	cases := []struct {
		name   string
		row    int
		target func(app *App) any
	}{
		{"request", rowSearch, func(app *App) any { return findReq(app, "Search Repos") }},
		{"folder", rowUser, func(app *App) any { return app.col().Folders[0] }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			a.step(core.Input{X: 100, Y: treeRowY(rowInfo), Click: true, Held: true})
			a.step(core.Input{X: 100, Y: treeRowY(tc.row), Held: true})
			if a.dragNode != model.Of(findReq(a, "Repository Info")) {
				t.Fatalf("dragging %v, want Repository Info", a.dragNode)
			}
			if want := tc.target(a); a.dropTarget != model.Of(want) {
				t.Fatalf("drop target %v, want %v", a.dropTarget, want)
			}
		})
	}
}
