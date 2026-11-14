package settingsui

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Widths the list of pages is pinned at: the dialog's own and one too
// narrow for the longest label.
const (
	navWideW   = DialogNavW
	navNarrowW = 60
)

// navList is the list of pages as the dialog shows it with page open
// open.
func navList(dialog *SettingsDialog, open string) *El {
	return Col(For(pages, func(_ int, p page) Node { return dialog.navItem(p, p.id == open) }))
}

// The list of pages, one open and the rest closed, is pinned at both
// widths.
func TestTheSettingsNavLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", navWideW},
		{"narrow", navNarrowW},
	}
	for _, size := range sizes {
		name := "nav-colors-open-" + size.name
		t.Run(name, func(t *testing.T) {
			var dialog SettingsDialog
			root := navList(&dialog, "colors")
			ctx, err := core.NewCtx(testTheme())
			if err != nil {
				t.Fatal(err)
			}
			Layout(ctx, root, size.width, len(pages)*core.RowH)
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

// Clicking a page in the list opens it at its top.
func TestClickingAPageInTheNavOpensItAtTheTop(t *testing.T) {
	dialog := SettingsDialog{Cat: "appearance"}
	dialog.scroll.Y = 120
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	root := navList(&dialog, dialog.Cat)
	Layout(ctx, root, navWideW, len(pages)*core.RowH)
	found := findAll(root, `text="Requests"`)
	if len(found) != 1 {
		t.Fatalf("%d rows show Requests:\n%s", len(found), Dump(root))
	}
	rect := found[0].Rect()
	ctx.In = core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	Update(ctx, root)
	if dialog.Cat != "requests" || dialog.scroll.Y != 0 {
		t.Fatalf("page %q at scroll %d, want requests at 0", dialog.Cat, dialog.scroll.Y)
	}
}
