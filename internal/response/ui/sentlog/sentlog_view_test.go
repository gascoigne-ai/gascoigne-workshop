package sentlog

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Sizes the list is pinned at: the sidebar's default width, and one so
// narrow that paths and URLs are cut.
const (
	wideW, wideH     = 260, 400
	narrowW, narrowH = 120, 200
)

const longPath = "Very Long Folder Name/Nested Folder/A Request Whose Name Does Not Fit"

func newCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// record adds a send of path that ended with status (0 = failed).
func record(log *Log, host Host, path, method, url string, status int) {
	log.Record(host, clock, path, &model.Request{Protocol: "rest", Method: method, URL: url}, status)
}

// threeSends is a log with a plain row, a failed one and a long one, oldest
// first, so the long one is on top.
func threeSends(log *Log, host Host) {
	record(log, host, "Users/List", "GET", "https://one.test/users", 200)
	record(log, host, "Orders/Create", "POST", "https://two.test/orders", 0)
	record(log, host, longPath, "GET", "https://three.test/a/very/long/url/that/does/not/fit/the/sidebar?page=2", 404)
}

func manySends(log *Log, host Host) {
	for i := range 30 {
		record(log, host, fmt.Sprintf("Folder/Request %d", i), "GET", fmt.Sprintf("https://many.test/%d", i), 200)
	}
}

// The laid-out list is pinned as text (testdata/*.dump, captured from the
// Go-built view), so a change in how it is built cannot move a pixel
// unnoticed.
func TestTheSentListLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		filter        string
		prepare       func(log *Log, host Host)
	}{
		{"empty-wide", wideW, wideH, "", nil},
		{"empty-narrow", narrowW, narrowH, "", nil},
		{"entries-wide", wideW, wideH, "", threeSends},
		{"entries-narrow", narrowW, narrowH, "", threeSends},
		{"filtered-wide", wideW, wideH, "orders", threeSends},
		{"filtered-out-wide", wideW, wideH, "nothing like it", threeSends},
		{"many-narrow", narrowW, narrowH, "", manySends},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var log Log
			host := &fakeHost{}
			if tc.prepare != nil {
				tc.prepare(&log, host)
			}
			root := log.List(host, tc.filter)
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

// listDriver runs the list frame by frame as the application does:
// rebuild the view, lay it out, deliver the input.
type listDriver struct {
	t      *testing.T
	ctx    *core.Ctx
	log    Log
	host   *fakeHost
	filter string
}

func newDriver(t *testing.T) *listDriver {
	return &listDriver{t: t, ctx: newCtx(t), host: &fakeHost{}}
}

// step runs one frame; input is built from the frame's laid-out view.
func (driver *listDriver) step(input func(root *El) core.Input) {
	root := driver.log.List(driver.host, driver.filter)
	Layout(driver.ctx, root, wideW, narrowH)
	driver.ctx.In = input(root)
	Update(driver.ctx, root)
}

// clickRow clicks the middle of the index-th row on screen.
func (driver *listDriver) clickRow(index int) {
	driver.t.Helper()
	driver.step(func(root *El) core.Input {
		rows := findAll(root, "control=row")
		if len(rows) <= index {
			driver.t.Fatalf("%d rows, want index %d:\n%s", len(rows), index, Dump(root))
		}
		rect := rows[index].Rect()
		return core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	})
}

func TestClickingARowOpensItsRequest(t *testing.T) {
	driver := newDriver(t)
	threeSends(&driver.log, driver.host)
	driver.clickRow(1)
	if len(driver.host.opened) != 1 || driver.host.opened[0] != "Orders/Create" {
		t.Fatalf("opened %v, want Orders/Create", driver.host.opened)
	}
}

func TestClickingAFilteredRowOpensThatRow(t *testing.T) {
	driver := newDriver(t)
	threeSends(&driver.log, driver.host)
	driver.filter = "users"
	driver.clickRow(0)
	if len(driver.host.opened) != 1 || driver.host.opened[0] != "Users/List" {
		t.Fatalf("opened %v, want Users/List", driver.host.opened)
	}
}

// After the wheel scrolls the list down, a click lands on the row drawn
// under the pointer, not on the one that was there before.
func TestAClickAfterScrollingOpensTheRowUnderThePointer(t *testing.T) {
	driver := newDriver(t)
	manySends(&driver.log, driver.host)
	var rowY int
	driver.step(func(root *El) core.Input {
		rect := findAll(root, "control=row")[0].Rect()
		rowY = rect.Y + rect.H/2
		return core.Input{X: wideW / 2, Y: rowY, Wheel: -3}
	})
	driver.step(func(*El) core.Input { return core.Input{X: wideW / 2, Y: rowY, Click: true} })
	if len(driver.host.opened) != 1 || driver.host.opened[0] == "Folder/Request 29" {
		t.Fatalf("opened %v, want a row further down than the newest", driver.host.opened)
	}
}

func TestAFailedSendShowsFailedAsItsStatus(t *testing.T) {
	var log Log
	host := &fakeHost{}
	threeSends(&log, host)
	root := log.List(host, "orders")
	Layout(newCtx(t), root, wideW, wideH)
	if found := findAll(root, `text="failed"`); len(found) != 1 {
		t.Fatalf("%d failed labels:\n%s", len(found), Dump(root))
	}
}
