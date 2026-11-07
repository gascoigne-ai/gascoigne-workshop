package calltrace

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// fakeHost is a window of 1000x800 with no right-click menus.
type fakeHost struct{}

func (fakeHost) Size() (int, int)                   { return 1000, 800 }
func (fakeHost) TextMenu(*TextField, func()) Option { return func(*El) {} }

var clock = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

func TestTheLogKeepsOnlyTheNewestEntries(t *testing.T) {
	var log Log
	for i := range Limit + 50 {
		log.Info(clock, fmt.Sprintf("entry %d", i))
	}
	lines := log.Lines()
	if len(lines) != Limit {
		t.Fatalf("%d lines, want %d", len(lines), Limit)
	}
	if !strings.HasSuffix(lines[len(lines)-1], fmt.Sprintf("entry %d", Limit+49)) {
		t.Fatalf("the newest entry is missing, last line %q", lines[len(lines)-1])
	}
}

func TestTheFilterNarrowsAndClearEmpties(t *testing.T) {
	var log Log
	log.Info(clock, "→ GET https://one.test")
	log.Info(clock, "→ GET https://two.test")
	log.SetFilter("TWO")
	if lines := log.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "two.test") {
		t.Fatalf("filtered = %q", lines)
	}
	log.SetFilter("")
	log.Clear()
	if len(log.Lines()) != 0 {
		t.Fatal("clear left entries")
	}
}

func TestErrorsAreMarkedAsFailures(t *testing.T) {
	var log Log
	log.Info(clock, "fine")
	log.Error(clock, "broken")
	entries := log.Entries()
	if entries[0].IsErr || !entries[1].IsErr {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestTheDialogShowsOnlyWhileOpenAndListsTheEntries(t *testing.T) {
	var log Log
	log.Info(clock, "→ GET https://one.test")
	if log.View(fakeHost{}) != nil {
		t.Fatal("the dialog is shown while closed")
	}
	log.Open()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	root := log.View(fakeHost{})
	Layout(ctx, root, 1000, 800)
	if dump := Dump(root); !strings.Contains(dump, "Log") || !strings.Contains(dump, "Clear") {
		t.Fatalf("dialog:\n%s", dump)
	}
	if !log.Shown() {
		t.Fatal("Shown is false while open")
	}
	log.Close()
	if log.View(fakeHost{}) != nil || log.Shown() {
		t.Fatal("the dialog stayed after Close")
	}
}

func testTheme() core.Theme {
	return core.Theme{
		Colors:   map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff), core.Bg: core.RGB(0x101010)},
		FontSize: 14, SmallSize: 12, MonoSize: 13,
	}
}

// The title bar template renders exactly what the Go view it replaced did.
func TestTitleBarTemplateMatchesTheGoView(t *testing.T) {
	var log Log
	host := fakeHost{}
	goView := Row(H(titleH), PadX(16), Gap(8), AlignItems(AlignCenter), BorderB(core.BorderSoft),
		Text("Log", Bold(), W(40)),
		Field(&log.filter, H(28), Grow(1), Bg(core.Sunken), Placeholder("Filter"), host.TextMenu(&log.filter, nil)),
		Button("Clear", W(64), H(core.CompactH), OnClick(log.Clear), Tooltip("Forget the entries")),
		IconButton(CrossIcon(Grow(1)), W(core.IconBtnW), H(core.CompactH), OnClick(log.Close), Tooltip("Close (Esc)")),
	)
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	tplView := log.titleBar(host)
	Layout(ctx, goView, 800, titleH)
	Layout(ctx, tplView, 800, titleH)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}
