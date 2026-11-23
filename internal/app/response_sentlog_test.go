package app

import (
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestSendingARequestListsItInTheSentLogNewestFirst(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: filepath.Join(t.TempDir(), "state.json")}))
	server := jsonServer(t)
	var requests []*request.Request
	walkRequests(&a.col().Folder, "", func(_ string, req *request.Request) { requests = append(requests, req) })
	first, second := requests[0], requests[1]
	for _, target := range []*request.Request{first, second} {
		a.openRequest(target)
		tab := a.cur()
		a.pointTabAt(tab, server.URL)
		a.send(tab)
		waitForResponse(t, a, tab)
	}
	rows := a.side.Sent().Rows(a.side.Filter().Text)
	if len(rows) != 2 || rows[0].Path != a.requestPath(second) || rows[1].Path != a.requestPath(first) {
		t.Fatalf("sent rows = %+v", rows)
	}
	if rows[0].Status != 200 || rows[0].URL != server.URL {
		t.Fatalf("newest row = %+v", rows[0])
	}
}

func TestTheSentLogComesBackAfterARestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	first := testApp(t, memorySettings(), withOptions(Options{StatePath: statePath}))
	tab := first.cur()
	first.pointTabAt(tab, jsonServer(t).URL)
	first.send(tab)
	waitForResponse(t, first, tab)

	second := testApp(t, memorySettings(), withOptions(Options{StatePath: statePath}))
	if rows := second.side.Sent().Rows(second.side.Filter().Text); len(rows) != 1 || rows[0].Status != 200 {
		t.Fatalf("sent rows after a restart = %+v", rows)
	}
}

func TestTheSidebarFilterNarrowsTheSentLog(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: filepath.Join(t.TempDir(), "state.json")}))
	tab := a.cur()
	a.pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)

	a.side.Filter().Text = "no-such-thing"
	if rows := a.side.Sent().Rows(a.side.Filter().Text); len(rows) != 0 {
		t.Fatalf("rows for a non-matching filter: %+v", rows)
	}
	a.side.Filter().Text = a.cur().Req.Name[:2]
	if rows := a.side.Sent().Rows(a.side.Filter().Text); len(rows) != 1 {
		t.Fatalf("rows for a matching filter: %+v", rows)
	}
}

func TestPickingASentRowOpensItsRequest(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: filepath.Join(t.TempDir(), "state.json")}))
	tab := a.cur()
	a.pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	row := a.side.Sent().Rows(a.side.Filter().Text)[0]

	a.openRequest(a.col().Folders[0].Requests[0]) // move away
	a.side.Sent().Open(a, row)
	if a.cur().Req != tab.Req {
		t.Fatalf("opened %q, want %q", a.cur().Req.Name, tab.Req.Name)
	}
}

func TestPickingASentRowOfADeletedRequestSaysSo(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: ""}))
	a.side.Sent().Open(a, history.Sent{Path: "Gone/Request"})
	if a.notices.Status() == "Ready" {
		t.Fatal("no note for a request that is gone")
	}
}

func TestTheSidebarCanShowTheSentLogAndTheEmptyNote(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: filepath.Join(t.TempDir(), "state.json")}))
	a.side.SetShowSent(true)
	a.Step(core.Input{}) // empty log
	tab := a.cur()
	a.pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	a.Step(core.Input{}) // with a row
	if !a.side.ShowSent() {
		t.Fatal("the sidebar left the log by itself")
	}
}
