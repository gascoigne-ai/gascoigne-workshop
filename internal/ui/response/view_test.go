package response

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// responseState is a named look of the response pane and how to give
// the open tab that look.
type responseState struct {
	name  string
	setUp func(t *testing.T, world *world)
}

// responseStates put the open tab in each look the response pane has:
// one per view mode, wrapped lines, the find bar open, earlier responses
// to pick from and an image body.
var responseStates = []responseState{
	{"compact", func(t *testing.T, world *world) { world.pane.SetMode(world.tab, viewer.Compact) }},
	{"pretty", func(t *testing.T, world *world) { world.pane.SetMode(world.tab, viewer.Pretty) }},
	{"raw", func(t *testing.T, world *world) { world.pane.SetMode(world.tab, viewer.Raw) }},
	{"wrap", func(t *testing.T, world *world) { world.pane.toggleWrap(world.tab) }},
	{"find", func(t *testing.T, world *world) {
		world.pane.OpenFind()
		world.pane.Field(world.tab).Insert("id")
		world.pane.runFind(world.tab)
	}},
	{"history", func(t *testing.T, world *world) {
		world.respond(request.Response{Status: 201, StatusText: "Created", Body: []byte(responseBody)})
	}},
	{"picture", func(t *testing.T, world *world) {
		world.respond(request.Response{Status: 200, StatusText: "OK", Body: pngBody(t)})
	}},
}

// responseIn is a world whose open tab has responseBody in the look named
// state.
func responseIn(t *testing.T, state string) *world {
	t.Helper()
	world := newWorld(t)
	world.respond(request.Response{Status: 200, StatusText: "OK", Body: []byte(responseBody)})
	for _, candidate := range responseStates {
		if candidate.name == state {
			candidate.setUp(t, world)
			return world
		}
	}
	t.Fatalf("no response state %q", state)
	return world
}

// The response pane is pinned as text (element kinds, options, handlers,
// rectangles) at two widths in every look, so moving it to markup cannot
// change a pixel unnoticed.
func TestTheResponsePaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", paneWideW},
		{"narrow", paneNarrowW},
	}
	for _, state := range responseStates {
		for _, size := range sizes {
			name := "response-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := responseIn(t, state.name)
				assertDumpPinned(t, world.ctx, world.body(), name, size.width, paneH)
			})
		}
	}
}

// The code view shows the tab's own view and the picture the tab's
// image: state Dump does not print, read from the widgets themselves.
func TestTheResponseWidgetsShowTheTabsState(t *testing.T) {
	world := responseIn(t, "compact")
	code := findWidget(world.body(), "*widget.codeB")
	if code == nil {
		t.Fatal("the response pane has no code view")
	}
	if got, want := fmt.Sprintf("%+v", code.Behavior()), fmt.Sprintf("view:%p", world.pane.Code(world.tab)); !strings.Contains(got, want) {
		t.Errorf("the code view lacks %q: %s", want, got)
	}
	world = responseIn(t, "picture")
	picture := findWidget(world.body(), "widget.imageB")
	if picture == nil {
		t.Fatal("the picture pane has no image")
	}
	if got, want := fmt.Sprintf("%+v", picture.Behavior()), fmt.Sprintf("img:%p", world.pane.state(world.tab).picture); !strings.Contains(got, want) {
		t.Errorf("the image lacks %q: %s", want, got)
	}
}

// The response headers are the key-value table in its read-only look, with
// the column widths the key-value tables keep; pinned as text at two widths.
func TestTheResponseHeadersTableLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", tableWideW},
		{"narrow", tableNarW},
	}
	for _, size := range sizes {
		name := "tableview-response-headers-" + size.name
		t.Run(name, func(t *testing.T) {
			world := newWorld(t)
			world.tab.Resp = &request.Response{Headers: []request.KV{
				{Name: "Content-Type", Value: "text/plain; charset=utf-8", Enabled: true},
				{Name: "Content-Length", Value: "82", Enabled: true},
			}}
			root := Stack(world.pane.View(world.ctx, world.tab, Headers, &widget.ScrollState{}))
			assertDumpPinned(t, world.ctx, root, name, size.width, paneH)
		})
	}
}

// The headers table keeps its column widths in the record the host gives,
// which is shared by every tab.
func TestTheResponseHeadersTableUsesTheHostsColumnWidths(t *testing.T) {
	world := newWorld(t)
	world.tab.Resp = &request.Response{Headers: []request.KV{{Name: "A", Value: "1", Enabled: true}}}
	root := world.pane.View(world.ctx, world.tab, Headers, &widget.ScrollState{})
	table := findWidget(root, "widget.colHandlesB")
	if table == nil {
		t.Fatal("the headers pane has no table")
	}
	if got, want := fmt.Sprintf("%+v", table.Behavior()), fmt.Sprintf("%p", &world.fake.widths); !strings.Contains(got, want) {
		t.Fatalf("the table lacks the host's widths %s: %s", want, got)
	}
}

// The row menu of the headers copies a row, or all the headers; it offers
// nothing that edits.
func TestTheResponseHeaderRowMenuCopies(t *testing.T) {
	world := newWorld(t)
	rows := []request.KV{{Name: "A", Value: "1", Enabled: true}, {Name: "B", Value: "2", Enabled: true}}
	clip := &clipboard{}
	world.ctx.Clipboard = clip
	items := copyRowItems(world.ctx, rows, 1)
	want := []string{"Copy value", "Copy name", "Copy as Name: Value", "Copy all headers"}
	if len(items) != len(want) {
		t.Fatalf("menu %+v, want %v", items, want)
	}
	for i, label := range want {
		if items[i].Label != label {
			t.Fatalf("item %d is %q, want %q", i, items[i].Label, label)
		}
	}
	for i, wantText := range []string{"2", "B", "B: 2", "A: 1\nB: 2\n"} {
		items[i].Run()
		if clip.text != wantText {
			t.Fatalf("%s copied %q, want %q", want[i], clip.text, wantText)
		}
	}
	if copyRowItems(world.ctx, rows, 2) != nil {
		t.Fatal("a row that is not there has no menu")
	}
}

// pinnedTimeline is a response whose every phase took time, so the note
// has all its lines; truncated adds the closing remark.
func pinnedTimeline(truncated bool) *request.Response {
	return &request.Response{
		Status: 200, StatusText: "OK", Duration: 412 * time.Millisecond, Size: 160430, Truncated: truncated,
		Timing: request.Timing{DNS: 3 * time.Millisecond, Connect: 11 * time.Millisecond, TLS: 48 * time.Millisecond,
			FirstByte: 290 * time.Millisecond, Transfer: 60 * time.Millisecond},
	}
}

// The note pane is pinned as text for a whole timeline and one that ends
// with the truncation remark, at two widths.
func TestTheNotePaneLaysOutAsPinned(t *testing.T) {
	states := []struct {
		name      string
		truncated bool
	}{
		{"timeline", false},
		{"truncated", true},
	}
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", tableWideW},
		{"narrow", tableNarW},
	}
	for _, state := range states {
		for _, size := range sizes {
			name := "notepane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t)
				world.tab.Resp = pinnedTimeline(state.truncated)
				var scroll widget.ScrollState
				assertDumpPinned(t, world.ctx, world.pane.View(world.ctx, world.tab, Timeline, &scroll), name, size.width, noteH)
			})
		}
	}
}

// What Dump does not show of the note pane: it scrolls with the scroll
// state the pane is given.
func TestTheNotePaneScrollsWithTheGivenState(t *testing.T) {
	world := newWorld(t)
	world.tab.Resp = pinnedTimeline(false)
	var scroll widget.ScrollState
	found := findWidget(world.pane.View(world.ctx, world.tab, Timeline, &scroll), "*widget.scrollB")
	if found == nil {
		t.Fatal("the note pane has no scroll")
	}
	if settings, want := fmt.Sprintf("%+v", found.Behavior()), fmt.Sprintf("%p", &scroll); !strings.Contains(settings, want) {
		t.Fatalf("the scroll lacks the pane's state %s: %s", want, settings)
	}
}

// The Tests pane is the assertions component's: the component asks the host
// for it with the scroll it was given.
func TestTheTestsPaneIsTheHostsSlot(t *testing.T) {
	world := newWorld(t)
	root := world.pane.View(world.ctx, world.tab, Tests, &widget.ScrollState{})
	if !strings.Contains(Dump(root), "results") {
		t.Fatalf("the Tests pane is not what the host drew:\n%s", Dump(root))
	}
}

func TestAnImageResponseIsShownAsAPicture(t *testing.T) {
	world := newWorld(t)
	world.respond(request.Response{Status: 200, StatusText: "OK", Body: pngBody(t),
		Headers: []request.KV{{Name: "Content-Type", Value: "text/plain", Enabled: true}}}) // mislabeled on purpose
	state := world.pane.state(world.tab)
	if state.picture == nil || !strings.HasPrefix(state.pictureInfo, "PNG 40x20") {
		t.Fatalf("picture %v info %q", state.picture, state.pictureInfo)
	}
	if msg, _ := world.pane.Empty(world.tab); msg != "" {
		t.Fatalf("a picture must not be reported as binary: %q", msg)
	}
	if d := Dump(world.body()); !strings.Contains(d, "PNG 40x20") || !strings.Contains(d, "Save") {
		t.Fatalf("pane:\n%s", d)
	}
	world.respond(request.Response{Status: 200, Body: []byte(`{"a":1}`)})
	if world.pane.state(world.tab).picture != nil {
		t.Fatal("the next, non-image response must clear the picture")
	}
}

func TestToggleWrapSwitchesTheResponseViewAndSurvivesANewResponse(t *testing.T) {
	world := newWorld(t)
	world.respond(request.Response{Status: 200, Body: []byte(`{"ok":true}`)})
	code := world.pane.Code(world.tab)
	if code.Wrap {
		t.Fatal("lines are not wrapped by default")
	}
	world.pane.toggleWrap(world.tab)
	if !code.Wrap {
		t.Fatal("toggling turns wrapping on")
	}
	world.pane.Refresh(world.tab) // a new response or view mode rebuilds the lines
	if !code.Wrap {
		t.Fatal("wrapping is a choice of the view, not of one response")
	}
	world.pane.toggleWrap(world.tab)
	if code.Wrap {
		t.Fatal("toggling again turns it off")
	}
}

// A body of megabytes was formatted, folded and searched on the UI
// goroutine and froze the window; now the view holds its first part and a
// note.
func TestAHugeBodyIsShownAsItsFirstPartWithANote(t *testing.T) {
	var body strings.Builder
	for body.Len() < 3<<20 {
		body.WriteString(`{"id": 1, "name": "row"}` + "\n")
	}
	world := newWorld(t)
	world.respond(request.Response{Status: 200, Body: []byte(body.String())})
	view := world.pane.Code(world.tab)
	if len(view.Lines) == 0 {
		t.Fatal("the view is empty")
	}
	last := lineText(view.Lines[len(view.Lines)-1])
	if !strings.Contains(last, "first 2.0") || !strings.Contains(last, "of 3.0") {
		t.Errorf("last line %q does not say the body is cut", last)
	}
	size := 0
	for _, line := range view.Lines {
		size += len(lineText(line))
	}
	if size > 2<<20+1<<10 {
		t.Errorf("the view holds %d bytes, want at most %d", size, 2<<20)
	}
	world.pane.runFind(world.tab) // the formatted-view paths must not read lines that were never built
	world.pane.CloseFind(world.tab)
}

func TestStatusPillsFollowTheDashboardBadges(t *testing.T) {
	for status, want := range map[int][2]core.Token{
		200: {core.OK, core.OKSoft}, 204: {core.OK, core.OKSoft},
		301: {core.Warn, core.WarnSoft}, 404: {core.Error, core.ErrorSoft}, 500: {core.Error, core.ErrorSoft},
	} {
		if fg, bg := statusPill(status); fg != want[0] || bg != want[1] {
			t.Errorf("%d: %v on %v, want %v on %v", status, fg, bg, want[0], want[1])
		}
	}
	for _, id := range []string{"dark", "light"} {
		th := theme.ThemeFor(id, theme.FontSizes{})
		for _, status := range []int{200, 301, 404} {
			fg, bg := statusPill(status)
			if r := core.ContrastRatio(th.Color(fg), th.Color(bg)); r < 3.5 {
				t.Errorf("%s: pill %d text %.2f:1 on its background", id, status, r)
			}
		}
	}
}

// What replaces the Body pane: a picture nor a text body say nothing, a body
// that is not text says its size.
func TestEmptyNamesABodyThatIsNotText(t *testing.T) {
	cases := []struct {
		name string
		resp *request.Response
		want string
	}{
		{"no response", nil, ""},
		{"text", &request.Response{Status: 200, Body: []byte(`{"a":1}`)}, ""},
		{"binary", &request.Response{Status: 200, Body: []byte{0, 1, 2, 3, 0, 0, 0}}, "Binary body, 7B (not shown)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t)
			world.tab.Resp = tc.resp
			if msg, isErr := world.pane.Empty(world.tab); msg != tc.want || isErr {
				t.Fatalf("Empty = %q, %v, want %q", msg, isErr, tc.want)
			}
		})
	}
}

// StatusView shows the answer only: the code, its text, the time and the size.
func TestStatusViewShowsTheAnswerOnly(t *testing.T) {
	cases := []struct {
		name string
		resp *request.Response
		want []string // nil means no pill
	}{
		{"ok", &request.Response{Status: 200, StatusText: "OK", Size: 2048}, []string{"200 OK", "2.00KB"}},
		{"redirect", &request.Response{Status: 302, StatusText: "Found"}, []string{"302 Found"}},
		{"not found", &request.Response{Status: 404, StatusText: "Not Found"}, []string{"404 Not Found"}},
		{"error", &request.Response{Err: "refused"}, nil},
		{"none", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t)
			world.tab.Resp = tc.resp
			got := world.pane.StatusView(world.ctx, world.tab)
			if tc.want == nil {
				if got != nil {
					t.Fatalf("pill %s, want none", Dump(got))
				}
				return
			}
			if got == nil {
				t.Fatal("no pill")
			}
			for _, want := range tc.want {
				if !strings.Contains(Dump(got), want) {
					t.Fatalf("pill lacks %q:\n%s", want, Dump(got))
				}
			}
		})
	}
}

// The headers of a response are read, not edited: no checkboxes, no bulk
// link, no grips and no delete, and a click on a cell edits nothing.
func TestTheResponseHeadersTableHasNoCheckboxesAndNoEdit(t *testing.T) {
	world := newWorld(t)
	world.tab.Resp = &request.Response{Headers: []request.KV{{Name: "A", Value: "1", Enabled: true}}}
	root := world.pane.View(world.ctx, world.tab, Headers, &widget.ScrollState{})
	dump := Dump(root)
	for _, bad := range []string{"checkB", "Bulk edit", "Drag to move the row", "Delete row"} {
		if strings.Contains(dump, bad) {
			t.Errorf("the response headers show %q:\n%s", bad, dump)
		}
	}
	Layout(world.ctx, root, tableWideW, paneH)
	cell := findWidget(root, "*widget.fieldB")
	if cell != nil {
		t.Fatal("the response headers have a text field to edit a cell in")
	}
}
