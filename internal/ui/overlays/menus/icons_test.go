package menus

import (
	"image/color"
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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
		{"Move to...", "folder"}, {"Copy to...", "copy"}, {"Move up", "chevron-up"}, {"Move down", "chevron-down"},
		{"Save body...", "save"}, {"Export as Postman...", "save"},
		{"Open in browser", "external"}, {"Undo", "undo"}, {"Redo", "redo"},
		{"GET /users", ""}, {"", ""},
	}
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			if got := menuIcon(widget.MenuItem{Label: test.label}); got != test.want {
				t.Fatalf("icon = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAnItemsOwnIconWinsOverTheLabelTable(t *testing.T) {
	if got := menuIcon(widget.MenuItem{Label: "Copy", Icon: "search"}); got != "search" {
		t.Fatalf("icon = %q, want the item's own", got)
	}
}

func TestEveryMenuIconInTheTableExists(t *testing.T) {
	for prefix, name := range menuIcons {
		if !widget.HasIcon(name) {
			t.Errorf("%q maps to the missing icon %q", prefix, name)
		}
	}
}

func TestOpenFillsTheIconsFromTheLabels(t *testing.T) {
	ctx := testCtx(t)
	Open(ctx, 10, 10, []widget.MenuItem{{Label: "Copy"}, {Label: "GET /users"}})
	if got := widget.PopupMenu(ctx).Items[0].Icon; got != "copy" {
		t.Fatalf("Copy row icon = %q, want copy", got)
	}
	if got := widget.PopupMenu(ctx).Items[1].Icon; got != "" {
		t.Fatalf("a row with no match has icon %q, want none", got)
	}
	if dump := view.Dump(View(ctx)); !strings.Contains(dump, "icon=copy") {
		t.Fatalf("the menu does not draw the Copy icon:\n%s", dump)
	}
}

func itoa(count int) string { return strconv.Itoa(count) }

// testCtx is the context the menu is built with in tests: the default faces.
func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{
		Colors:   map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff), core.Bg: core.RGB(0x101010)},
		FontSize: 14, SmallSize: 12, MonoSize: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}
