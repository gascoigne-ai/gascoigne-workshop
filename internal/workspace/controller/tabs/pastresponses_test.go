package tabs

import (
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

var base = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

func response(status int) *model.Response { return &model.Response{Status: status} }

func TestRememberKeepsNewestFirstAndSkipsFailures(t *testing.T) {
	tab := New(newRequest())
	tab.Remember(nil, base)
	tab.Remember(&model.Response{Err: "boom"}, base)
	if len(tab.Earlier) != 0 {
		t.Fatalf("nothing to keep, got %d", len(tab.Earlier))
	}
	tab.Remember(response(200), base)
	tab.Remember(response(201), base.Add(time.Minute))
	if len(tab.Earlier) != 2 || tab.Earlier[0].Resp.Status != 201 {
		t.Fatalf("earlier = %+v", tab.Earlier)
	}
}

func TestRememberKeepsTheLastTen(t *testing.T) {
	tab := New(newRequest())
	for status := range PastLimit + 5 {
		tab.Remember(response(status), base)
	}
	if len(tab.Earlier) != PastLimit || tab.Earlier[0].Resp.Status != PastLimit+4 {
		t.Fatalf("len %d first %d", len(tab.Earlier), tab.Earlier[0].Resp.Status)
	}
}

func TestRecallSwapsTheShownResponseIntoItsPlace(t *testing.T) {
	tab := New(newRequest())
	tab.Resp = response(500 - 300)
	tab.Earlier = []PastResponse{{Resp: response(301)}, {Resp: response(302)}}
	if !tab.Recall(1, base) {
		t.Fatal("recall failed")
	}
	if tab.Resp.Status != 302 || tab.Earlier[1].Resp.Status != 200 || !tab.Earlier[1].At.Equal(base) {
		t.Fatalf("shown %d earlier %+v", tab.Resp.Status, tab.Earlier)
	}
}

func TestRecallWithoutAGoodShownResponseDropsTheEntry(t *testing.T) {
	tab := New(newRequest())
	tab.Resp = &model.Response{Err: "boom"}
	tab.Earlier = []PastResponse{{Resp: response(301)}, {Resp: response(302)}}
	tab.Recall(0, base)
	if tab.Resp.Status != 301 || len(tab.Earlier) != 1 || tab.Earlier[0].Resp.Status != 302 {
		t.Fatalf("shown %+v earlier %+v", tab.Resp, tab.Earlier)
	}
}

func TestRecallOutOfRangeChangesNothing(t *testing.T) {
	tab := New(newRequest())
	tab.Earlier = []PastResponse{{Resp: response(301)}}
	if tab.Recall(-1, base) || tab.Recall(1, base) || tab.Resp != nil {
		t.Fatal("an index outside the history recalls nothing")
	}
}
