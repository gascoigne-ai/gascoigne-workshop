package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
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
			if got := menuIcon(menuItem{label: test.label}); got != test.want {
				t.Fatalf("icon = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAnItemsOwnIconWinsOverTheLabelTable(t *testing.T) {
	if got := menuIcon(menuItem{label: "Copy", icon: "search"}); got != "search" {
		t.Fatalf("icon = %q, want the item's own", got)
	}
}

func TestEveryMenuIconInTheTableExists(t *testing.T) {
	for prefix, name := range menuIcons {
		if !vml.HasIcon(name) {
			t.Errorf("%q maps to the missing icon %q", prefix, name)
		}
	}
}

func TestAMenuRowDrawsItsIconAndKeepsLabelsAligned(t *testing.T) {
	a, _ := newTestApp(t)
	a.openMenu(10, 10, []menuItem{{label: "Copy"}, {label: "GET /users"}})
	withIcon := vml.Dump(a.menuRow(a.ctxMenu, 0, a.ctxMenu.items[0]))
	without := vml.Dump(a.menuRow(a.ctxMenu, 1, a.ctxMenu.items[1]))
	if !strings.Contains(withIcon, "icon=copy") {
		t.Fatalf("no icon on the Copy row:\n%s", withIcon)
	}
	if strings.Contains(without, "icon=") {
		t.Fatalf("an icon on a row with no match:\n%s", without)
	}
	if !strings.Contains(without, "w="+itoa(menuIconW)) {
		t.Fatalf("the unmatched row has no icon-sized spacer, so its label would not line up:\n%s", without)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
