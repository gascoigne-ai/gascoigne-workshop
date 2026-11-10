package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestMenuRowsGetAnIconFromTheirLabel(t *testing.T) {
	tests := []struct {
		label, want string
	}{
		{"Copy", "copy"}, {"Copy URL", "copy"}, {"Copy as cURL", "copy"}, {"Duplicate", "copy"},
		{"Cut", "scissors"}, {"Paste", "clipboard"}, {"Import cURL from clipboard", "clipboard"},
		{"Delete", "trash"}, {"Rename...", "pencil"}, {"Rename", "pencil"},
		{"Close others", "cross"}, {"Find", "search"},
		{"New request", "plus"}, {"New environment...", "plus"}, {"Insert row above", "plus"},
		{"Move up", "chevron-up"}, {"Move down", "chevron-down"},
		{"Save body...", "save"}, {"Export as Postman...", "save"},
		{"Open in browser", "external"}, {"Undo", "undo"}, {"Redo", "redo"},
		{"GET /users", ""}, {"", ""},
	}
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			if got := menuIcon(view.MenuItem{Label: test.label}); got != test.want {
				t.Fatalf("icon = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAnItemsOwnIconWinsOverTheLabelTable(t *testing.T) {
	if got := menuIcon(view.MenuItem{Label: "Copy", Icon: "search"}); got != "search" {
		t.Fatalf("icon = %q, want the item's own", got)
	}
}

func TestEveryMenuIconInTheTableExists(t *testing.T) {
	for prefix, name := range menuIcons {
		if !view.HasIcon(name) {
			t.Errorf("%q maps to the missing icon %q", prefix, name)
		}
	}
}

func TestOpenMenuFillsTheIconsFromTheLabels(t *testing.T) {
	a, _ := newTestApp(t)
	a.openMenu(10, 10, []view.MenuItem{{Label: "Copy"}, {Label: "GET /users"}})
	if got := a.ctxMenu.Items[0].Icon; got != "copy" {
		t.Fatalf("Copy row icon = %q, want copy", got)
	}
	if got := a.ctxMenu.Items[1].Icon; got != "" {
		t.Fatalf("a row with no match has icon %q, want none", got)
	}
	if dump := view.Dump(a.contextMenuView()); !strings.Contains(dump, "icon=copy") {
		t.Fatalf("the menu does not draw the Copy icon:\n%s", dump)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
