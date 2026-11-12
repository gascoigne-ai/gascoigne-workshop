package colorpage

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Window sizes the page is pinned at: the settings dialog's usual width,
// and one so narrow that labels and notes are cut.
const (
	wideW, wideH     = 900, 600
	narrowW, narrowH = 220, 400
)

func newCtx(t *testing.T) *w.Ctx {
	t.Helper()
	ctx, err := w.NewCtx(w.Theme{Colors: map[w.Token]color.RGBA{w.Text: w.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// refuse types text into the hex field of the second color and applies
// it, so that row shows a note and a red border.
func refuse(text string) func(page *Page, host *fakeHost) {
	return func(page *Page, host *fakeHost) {
		custom := appearance.CustomColors[1]
		page.Input(host, custom.Token).Text = text
		page.edited(host, custom)
	}
}

// The laid-out page is pinned as text (testdata/*.dump, captured from the
// Go-built view), so a change in how it is built cannot move a pixel
// unnoticed.
func TestTheColorPageLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		prepare       func(page *Page, host *fakeHost)
	}{
		{"default-wide", wideW, wideH, nil},
		{"default-narrow", narrowW, narrowH, nil},
		{"unreadable-wide", wideW, wideH, refuse("teal")},
		{"contrast-narrow", narrowW, narrowH, refuse("#000000")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var page Page
			host := newHost()
			if tc.prepare != nil {
				tc.prepare(&page, host)
			}
			root := page.View(host)
			Layout(newCtx(t), root, tc.width, tc.height)
			want, err := os.ReadFile(filepath.Join("testdata", tc.name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got := Dump(root); got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// findAll is every element under root, in tree order, whose own line of
// Dump has all the parts.
func findAll(root *El, parts ...string) []*El {
	var found []*El
	line, _, _ := strings.Cut(Dump(root), "\n")
	matches := true
	for _, part := range parts {
		matches = matches && strings.Contains(line, part)
	}
	if matches {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findAll(child, parts...)...)
	}
	return found
}

// pageDriver runs the page frame by frame as the application does:
// rebuild the view, lay it out, deliver the input.
type pageDriver struct {
	t    *testing.T
	ctx  *w.Ctx
	page Page
	host *fakeHost
}

func newDriver(t *testing.T) *pageDriver {
	return &pageDriver{t: t, ctx: newCtx(t), host: newHost()}
}

// step runs one frame; input is built from the frame's laid-out view.
func (driver *pageDriver) step(input func(root *El) w.Input) {
	root := driver.page.View(driver.host)
	Layout(driver.ctx, root, wideW, wideH)
	driver.ctx.In = input(root)
	Update(driver.ctx, root)
}

// clickOn clicks the middle of the index-th element matching parts.
func (driver *pageDriver) clickOn(index int, parts ...string) {
	driver.t.Helper()
	driver.step(func(root *El) w.Input {
		found := findAll(root, parts...)
		if len(found) <= index {
			driver.t.Fatalf("%d elements match %q, want index %d:\n%s", len(found), parts, index, Dump(root))
		}
		rect := found[index].Rect()
		return w.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	})
}

// retype replaces the text of the focused field the way a user does:
// go to the end, backspace over it, then type.
func (driver *pageDriver) retype(text string) {
	edits := []w.Edit{w.EditEnd}
	for range 16 {
		edits = append(edits, w.EditBackspace)
	}
	driver.step(func(*El) w.Input { return w.Input{X: 5000, Y: 5000, Edits: edits} })
	driver.step(func(*El) w.Input { return w.Input{X: 5000, Y: 5000, Typed: []rune(text)} })
}

func (driver *pageDriver) saved() map[string]string {
	return driver.host.current.Colors[appearance.ThemeIDOf(driver.host.current.Theme)]
}

// Typing a readable hex with enough contrast into the second row's field
// saves that color, and only that one.
func TestTypingAHexIntoARowSavesThatRowsColor(t *testing.T) {
	driver := newDriver(t)
	driver.clickOn(1, "fieldB")
	driver.retype("#ffffff")
	second := appearance.CustomColors[1].Token
	if saved := driver.saved(); len(saved) != 1 || saved[string(second)] != "#ffffff" {
		t.Fatalf("saved %+v, want only %s", saved, second)
	}
	if note := driver.page.Note(second); note != "" {
		t.Fatalf("note %q", note)
	}
}

func TestTypingARefusedHexShowsItsNoteUnderThatRow(t *testing.T) {
	cases := []struct{ name, typed, note string }{
		{"unreadable", "zz", "Use #rrggbb"},
		{"low contrast", "#000000", "Contrast 1.0:1 is below"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newDriver(t)
			driver.clickOn(1, "fieldB")
			driver.retype(tc.typed)
			second := appearance.CustomColors[1].Token
			if note := driver.page.Note(second); !strings.HasPrefix(note, tc.note) {
				t.Fatalf("note %q, want %q", note, tc.note)
			}
			if len(driver.saved()) != 0 {
				t.Fatalf("a refused color was saved: %+v", driver.saved())
			}
			root := driver.page.View(driver.host)
			Layout(driver.ctx, root, wideW, wideH)
			if bordered := findAll(root, "fieldB", "border=error"); len(bordered) != 1 || bordered[0] != findAll(root, "fieldB")[1] {
				t.Fatalf("only the second field should have the error border:\n%s", Dump(root))
			}
		})
	}
}

// A row's Reset drops that row's override and keeps the others.
func TestARowsResetDropsOnlyItsOverride(t *testing.T) {
	driver := newDriver(t)
	theme := appearance.ThemeIDOf(driver.host.current.Theme)
	first, second := appearance.CustomColors[0].Token, appearance.CustomColors[1].Token
	driver.host.current.Colors = map[string]map[string]string{theme: {string(first): "#111111", string(second): "#222222"}}
	driver.clickOn(1, `text="Reset"`)
	if saved := driver.saved(); len(saved) != 1 || saved[string(first)] != "#111111" {
		t.Fatalf("saved %+v, want only %s", saved, first)
	}
}

func TestResetColorsDropsEveryOverrideAndNote(t *testing.T) {
	driver := newDriver(t)
	theme := appearance.ThemeIDOf(driver.host.current.Theme)
	first := appearance.CustomColors[0].Token
	driver.host.current.Colors = map[string]map[string]string{theme: {string(first): "#111111"}}
	refuse("teal")(&driver.page, driver.host)
	driver.clickOn(0, `text="Reset colors"`)
	if len(driver.host.current.Colors) != 0 || driver.page.Note(appearance.CustomColors[1].Token) != "" {
		t.Fatalf("colors %+v note %q", driver.host.current.Colors, driver.page.Note(appearance.CustomColors[1].Token))
	}
}
