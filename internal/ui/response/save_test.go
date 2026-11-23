package response

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
)

func TestHistoryMenuListsEarlierResponsesNewestFirstAndSwapsOnPick(t *testing.T) {
	world := newWorld(t)
	for _, status := range []int{200, 201, 202} {
		world.respond(request.Response{Status: status, StatusText: "S", Body: []byte("{}")})
	}
	items := world.pane.historyMenu(world.tab)
	if len(items) != 2 || !strings.HasPrefix(items[0].Label, "201") {
		t.Fatalf("menu %+v", items)
	}
	items[1].Run() // pick the 200
	if world.tab.Resp.Status != 200 || world.tab.Earlier[1].Resp.Status != 202 {
		t.Fatalf("shown %d, history %+v", world.tab.Resp.Status, world.tab.Earlier)
	}
}

func TestSaveBodyAsksForANameAndReportsWhenThereIsNothingToSave(t *testing.T) {
	world := newWorld(t)
	world.pane.saveBody(world.tab) // no response yet
	if len(world.fake.picked) != 0 || len(world.fake.toasts) != 1 || world.fake.toasts[0].level != severity.Error {
		t.Fatalf("picked %v toasts %+v, want only the complaint", world.fake.picked, world.fake.toasts)
	}
	world.tab.Resp = &request.Response{Status: 200, Body: []byte(`{"a":1}`)}
	world.pane.saveBody(world.tab)
	if len(world.fake.picked) != 1 || world.fake.picked[0] != "Save response body|response.txt" {
		t.Fatalf("picked %v, want the title and the suggested name", world.fake.picked)
	}
}

func TestWriteBodySavesTheBodyAndSaysSo(t *testing.T) {
	world := newWorld(t)
	world.fake.home = "/home/user"
	world.tab.Resp = &request.Response{Status: 200, Body: []byte(`{"a":1}`)}
	world.pane.saveBody(world.tab)
	world.fake.pickedTo("  ~/out.json ") // the answer of the file chooser
	if len(world.fake.saved) != 1 || world.fake.saved[0].path != filepath.Join("/home/user", "out.json") || string(world.fake.saved[0].body) != `{"a":1}` {
		t.Fatalf("saved %+v, want the body in the home folder", world.fake.saved)
	}
	if len(world.fake.toasts) != 1 || world.fake.toasts[0].text != "Saved 7B to out.json" || world.fake.toasts[0].level != severity.Info {
		t.Fatalf("toasts %+v", world.fake.toasts)
	}
}

func TestAnEmptyPathSavesNothing(t *testing.T) {
	world := newWorld(t)
	world.tab.Resp = &request.Response{Status: 200, Body: []byte("x")}
	world.pane.writeBody(world.tab, "   ")
	if len(world.fake.saved) != 0 {
		t.Fatalf("saved %+v", world.fake.saved)
	}
}

func TestAFailedSaveIsReported(t *testing.T) {
	world := newWorld(t)
	world.fake.writeErr = errors.New("disk full")
	world.tab.Resp = &request.Response{Status: 200, Body: []byte("x")}
	world.pane.writeBody(world.tab, "/out/body.bin")
	if len(world.fake.toasts) != 1 || world.fake.toasts[0].text != "Saving failed: disk full" || world.fake.toasts[0].level != severity.Error {
		t.Fatalf("toasts %+v", world.fake.toasts)
	}
}

func TestOpenInBrowserPreviewsTheBodyWithItsContentType(t *testing.T) {
	cases := []struct {
		name    string
		resp    *request.Response
		preview error
		want    []string // content types handed to the preview
		toast   string
		level   severity.Level
	}{
		{"html", &request.Response{Status: 200, Body: []byte("<h1>hi</h1>"), Headers: []request.KV{{Name: "Content-Type", Value: "text/html"}}},
			nil, []string{"text/html"}, "Opened in the browser for 5 minutes", severity.Info},
		{"no content type", &request.Response{Status: 200, Body: []byte("plain")},
			nil, []string{"text/plain; charset=utf-8"}, "Opened in the browser for 5 minutes", severity.Info},
		{"no response", nil, nil, nil, "There is no response to open", severity.Error},
		{"failed response", &request.Response{Err: "refused"}, nil, nil, "There is no response to open", severity.Error},
		{"preview fails", &request.Response{Status: 200, Body: []byte("x")}, errors.New("no browser"),
			[]string{"text/plain; charset=utf-8"}, "Preview failed: no browser", severity.Error},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t)
			world.fake.preview = tc.preview
			world.tab.Resp = tc.resp
			world.pane.openInBrowser(world.tab)
			if strings.Join(world.fake.previewed, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("previewed %v, want %v", world.fake.previewed, tc.want)
			}
			if len(world.fake.toasts) != 1 || world.fake.toasts[0].text != tc.toast || world.fake.toasts[0].level != tc.level {
				t.Fatalf("toasts %+v, want %q", world.fake.toasts, tc.toast)
			}
		})
	}
}

func TestTheBodyMenuCopiesSavesAndFinds(t *testing.T) {
	world := withResponse(t, findBody)
	clip := &clipboard{}
	world.ctx.Clipboard = clip
	world.fake.curl = "curl https://api.example.test/users"
	items := world.pane.menuItems(world.ctx, world.tab)
	byLabel := map[string]int{}
	for i, item := range items {
		byLabel[item.Label] = i
	}
	for _, label := range []string{"Copy", "Copy body", "Save body...", "Open in browser", "Copy as cURL", "Select all", "Find"} {
		if _, ok := byLabel[label]; !ok {
			t.Fatalf("menu lacks %q: %+v", label, items)
		}
	}
	if !items[byLabel["Copy"]].Disabled || items[byLabel["Save body..."]].Disabled {
		t.Fatal("Copy waits for a selection, Save for nothing but a response")
	}
	if got := items[byLabel["Find"]].Shortcut; got != "Ctrl+F" {
		t.Fatalf("Find shortcut %q, want the host's", got)
	}
	items[byLabel["Copy body"]].Run()
	if clip.text != findBody {
		t.Fatalf("copied %q", clip.text)
	}
	items[byLabel["Copy as cURL"]].Run()
	if clip.text != "curl https://api.example.test/users" {
		t.Fatalf("copied %q", clip.text)
	}
	items[byLabel["Select all"]].Run()
	if !world.pane.Code(world.tab).Selected() {
		t.Fatal("Select all selects the lines")
	}
	items[byLabel["Find"]].Run()
	if !world.pane.Look(world.tab).FindOpen {
		t.Fatal("Find opens the bar")
	}
}

func TestTheMenuOfAFailedResponseCannotSaveOrOpenIt(t *testing.T) {
	world := newWorld(t)
	world.tab.Resp = &request.Response{Err: "refused"}
	for _, item := range world.pane.menuItems(world.ctx, world.tab) {
		if (item.Label == "Save body..." || item.Label == "Open in browser") && !item.Disabled {
			t.Fatalf("%q must be disabled without a body", item.Label)
		}
	}
}
