package ui

import (
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestSendingARequestListsItInTheSentLogNewestFirst(t *testing.T) {
	a := stateApp(t, filepath.Join(t.TempDir(), "state.json"))
	server := jsonServer(t)
	var requests []*model.Request
	walkRequests(&a.col.Folder, "", func(_ string, req *model.Request) { requests = append(requests, req) })
	first, second := requests[0], requests[1]
	for _, target := range []*model.Request{first, second} {
		a.openRequest(target)
		tab := a.cur()
		pointTabAt(tab, server.URL)
		a.send(tab)
		waitForResponse(t, a, tab)
	}
	rows := a.sent.Rows(a.filter.Text)
	if len(rows) != 2 || rows[0].Path != a.requestPath(second) || rows[1].Path != a.requestPath(first) {
		t.Fatalf("sent rows = %+v", rows)
	}
	if rows[0].Status != 200 || rows[0].URL != server.URL {
		t.Fatalf("newest row = %+v", rows[0])
	}
}

func TestTheSentLogComesBackAfterARestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	first := stateApp(t, statePath)
	tab := first.cur()
	pointTabAt(tab, jsonServer(t).URL)
	first.send(tab)
	waitForResponse(t, first, tab)

	second := stateApp(t, statePath)
	if rows := second.sent.Rows(second.filter.Text); len(rows) != 1 || rows[0].Status != 200 {
		t.Fatalf("sent rows after a restart = %+v", rows)
	}
}

func TestTheSidebarFilterNarrowsTheSentLog(t *testing.T) {
	a := stateApp(t, filepath.Join(t.TempDir(), "state.json"))
	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)

	a.filter.Text = "no-such-thing"
	if rows := a.sent.Rows(a.filter.Text); len(rows) != 0 {
		t.Fatalf("rows for a non-matching filter: %+v", rows)
	}
	a.filter.Text = a.cur().req.Name[:2]
	if rows := a.sent.Rows(a.filter.Text); len(rows) != 1 {
		t.Fatalf("rows for a matching filter: %+v", rows)
	}
}

func TestPickingASentRowOpensItsRequest(t *testing.T) {
	a := stateApp(t, filepath.Join(t.TempDir(), "state.json"))
	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	row := a.sent.Rows(a.filter.Text)[0]

	a.openRequest(a.col.Folders[0].Requests[0]) // move away
	a.sent.Open(a, row)
	if a.cur().req != tab.req {
		t.Fatalf("opened %q, want %q", a.cur().req.Name, tab.req.Name)
	}
}

func TestPickingASentRowOfADeletedRequestSaysSo(t *testing.T) {
	a := stateApp(t, "")
	a.sent.Open(a, history.Sent{Path: "Gone/Request"})
	if a.status == "Ready" {
		t.Fatal("no note for a request that is gone")
	}
}

func TestTheSidebarCanShowTheSentLogAndTheEmptyNote(t *testing.T) {
	a := stateApp(t, filepath.Join(t.TempDir(), "state.json"))
	a.showSent = true
	a.step(core.Input{}) // empty log
	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	a.step(core.Input{}) // with a row
	if !a.showSent {
		t.Fatal("the sidebar left the log by itself")
	}
}
