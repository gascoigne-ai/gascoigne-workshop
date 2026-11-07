package sentlog

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/adapter/history"
	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// fakeHost keeps what the log asks the application to do.
type fakeHost struct {
	dir     string
	toasts  []string
	status  string
	opened  []string
	present map[string]bool
}

func (h *fakeHost) HistoryDir() string          { return h.dir }
func (h *fakeHost) Toast(text string, _ bool)   { h.toasts = append(h.toasts, text) }
func (h *fakeHost) SetStatus(text string)       { h.status = text }
func (h *fakeHost) Badge(_, method string) Node { return Text(method) }
func (h *fakeHost) OpenRequest(path string) bool {
	h.opened = append(h.opened, path)
	return h.present[path]
}

var clock = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

func send(log *Log, host Host, path, url string) {
	req := &model.Request{Protocol: "rest", Method: "GET", URL: url}
	log.Record(host, clock, path, req, 200)
}

func TestRecordedSendsListNewestFirstAndFilterByPathMethodOrURL(t *testing.T) {
	var log Log
	host := &fakeHost{}
	send(&log, host, "Users/List", "https://one.test/users")
	send(&log, host, "Orders/Create", "https://two.test/orders")
	rows := log.Rows("")
	if len(rows) != 2 || rows[0].Path != "Orders/Create" {
		t.Fatalf("rows = %+v", rows)
	}
	for filter, want := range map[string]string{"USERS": "Users/List", "two.test": "Orders/Create", "get": ""} {
		got := log.Rows(filter)
		switch {
		case want == "" && len(got) != 2:
			t.Errorf("%q kept %d rows, want both", filter, len(got))
		case want != "" && (len(got) != 1 || got[0].Path != want):
			t.Errorf("%q kept %+v, want only %s", filter, got, want)
		}
	}
}

func TestTheLogKeepsOnlyTheNewestSentLimitEntries(t *testing.T) {
	var log Log
	host := &fakeHost{}
	for i := range history.SentLimit + 5 {
		send(&log, host, fmt.Sprintf("R%d", i), "https://x.test")
	}
	if rows := log.Rows(""); len(rows) != history.SentLimit || rows[0].Path != fmt.Sprintf("R%d", history.SentLimit+4) {
		t.Fatalf("%d rows, first %q", len(rows), rows[0].Path)
	}
}

func TestEntriesSurviveAReloadFromTheStateFolder(t *testing.T) {
	host := &fakeHost{dir: t.TempDir()}
	var first Log
	send(&first, host, "Users/List", "https://one.test")
	var second Log
	second.Load(host)
	if rows := second.Rows(""); len(rows) != 1 || rows[0].Path != "Users/List" {
		t.Fatalf("rows after a reload = %+v", rows)
	}
}

func TestOpeningARowOfARequestThatIsGoneSaysSoInTheStatusLine(t *testing.T) {
	var log Log
	host := &fakeHost{present: map[string]bool{"Here": true}}
	log.Open(host, history.Sent{Path: "Here"})
	if host.status != "" || len(host.opened) != 1 {
		t.Fatalf("status %q opened %v", host.status, host.opened)
	}
	log.Open(host, history.Sent{Path: "Gone"})
	if !strings.Contains(host.status, "Gone") {
		t.Fatalf("status %q", host.status)
	}
}

func TestTheListShowsAnEmptyNoteAndTheRows(t *testing.T) {
	var log Log
	host := &fakeHost{}
	if dump := Dump(log.List(host, "")); !strings.Contains(dump, "Nothing sent yet") {
		t.Fatalf("empty list:\n%s", dump)
	}
	send(&log, host, "Users/List", "https://one.test")
	if dump := Dump(log.List(host, "")); !strings.Contains(dump, "Users/List") || !strings.Contains(dump, "200") {
		t.Fatalf("list:\n%s", dump)
	}
}

// The row template renders exactly what the Go view it replaced did.
func TestRowTemplateMatchesTheGoView(t *testing.T) {
	var log Log
	host := &fakeHost{}
	sent := history.Sent{Path: "Users/List", URL: "https://one.test", Protocol: "rest", Method: "GET", Status: 200, At: clock}
	goView := Col(PadX(core.Gutter), PadY(4), HoverBg(core.Hover), Control("row"), OnClick(func() { log.Open(host, sent) }),
		Row(Gap(6), AlignItems(AlignBaseline),
			host.Badge(sent.Protocol, sent.Method),
			Text(sent.Path, Ellipsis(), Grow(1)),
			Text(status(sent.Status), Small(), Muted()),
		),
		Row(Text(sent.URL+"  "+sent.At.Format("15:04:05"), Small(), Muted(), Ellipsis(), Grow(1))),
	)
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	tplView := log.row(host, sent)
	Layout(ctx, goView, 300, 60)
	Layout(ctx, tplView, 300, 60)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}
