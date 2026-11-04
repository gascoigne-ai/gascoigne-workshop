package assert

import (
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func resp() *model.Response {
	return &model.Response{
		Status: 200, Duration: 120 * time.Millisecond,
		Headers: []model.KV{{Name: "Content-Type", Value: "application/json; charset=utf-8", Enabled: true}, {Name: "X-Id", Value: "7", Enabled: true}},
		Body:    []byte(`{"total_count":116,"ok":true,"none":null,"name":"alpha","items":[{"name":"a","tags":["x","y"]},{"name":"b"}]}`),
	}
}

func TestAssertions(t *testing.T) {
	tests := []struct {
		line string
		ok   bool
		note string // a substring of the detail when it fails
	}{
		{"status == 200", true, ""},
		{"status != 404", true, ""},
		{"status < 300", true, ""},
		{"status >= 201", false, "got 200"},
		{"status 200", false, "operator"},
		{"status == ok", false, "not a number"},
		{"time < 500ms", true, ""},
		{"time < 100ms", false, "got 120ms"},
		{"time == 1s", false, "operator"},
		{"time < soon", false, "not a duration"},
		{"header Content-Type contains json", true, ""},
		{"header content-type == \"nope\"", false, "got"},
		{"header X-Id exists", true, ""},
		{"header X-Missing exists", false, "not there"},
		{"header X-Id == 7", true, ""},
		{`body contains "alpha"`, true, ""},
		{`body contains "zzz"`, false, "got"},
		{"json total_count > 100", true, ""},
		{"json total_count == 116", true, ""},
		{"json total_count < 100", false, "got 116"},
		{`json name == "alpha"`, true, ""},
		{"json name contains lph", true, ""},
		{"json ok == true", true, ""},
		{"json none == null", true, ""},
		{"json ok != true", false, "got true"},
		{`json items[0].name == "a"`, true, ""},
		{"json items.length == 2", true, ""},
		{"json items[0].tags.length >= 2", true, ""},
		{"json items[0].tags contains y", true, ""},
		{"json items[0].tags contains z", false, "does not contain"},
		{"json items[5].name exists", false, "no such path"},
		{"json items[1].tags exists", false, "no such path"},
		{"json name.length == 5", true, ""},
		{"json items == 1", false, "list"},
		{"json missing == 1", false, "no such path"},
		{"json total_count contains 1", false, "string or a list"},
		{"jsn total_count > 1", false, "unknown subject"},
	}
	for _, tt := range tests {
		res := Run(tt.line, resp())
		if len(res) != 1 {
			t.Fatalf("%q: %d results", tt.line, len(res))
		}
		if res[0].OK != tt.ok || (tt.note != "" && !strings.Contains(res[0].Detail, tt.note)) {
			t.Errorf("%q: ok=%v detail=%q, want ok=%v with %q", tt.line, res[0].OK, res[0].Detail, tt.ok, tt.note)
		}
	}
}

func TestRunSkipsCommentsAndBlankLinesAndSummarizes(t *testing.T) {
	res := Run("# checks\n\nstatus == 200\n  status == 500  \n", resp())
	if len(res) != 2 || !res[0].OK || res[1].OK || res[1].Line != "status == 500" {
		t.Fatalf("%+v", res)
	}
	if Summary(res) != "1/2 passed" || Summary(nil) != "" {
		t.Fatalf("summary %q", Summary(res))
	}
}

func TestNoResponseOrAFailedOneFailsEveryAssertion(t *testing.T) {
	for _, r := range []*model.Response{nil, {Err: "boom"}} {
		res := Run("status == 200", r)
		if len(res) != 1 || res[0].OK || !strings.Contains(res[0].Detail, "no response") {
			t.Fatalf("%+v", res)
		}
	}
	if res := Run("json a == 1", &model.Response{Status: 200, Body: []byte("not json")}); res[0].OK || !strings.Contains(res[0].Detail, "not JSON") {
		t.Fatalf("%+v", res)
	}
}

func TestSetLinesCaptureValuesForLaterRequests(t *testing.T) {
	r := resp()
	text := "set count = json total_count\nset first = json items[0].name\nset ct = header content-type\nset code = status\nset tags = json items[0].tags\nset none = json none\nset missing = json nope\nset bad\nset x = body"
	caps := Captures(text, r)
	got := map[string]string{}
	for _, c := range caps {
		got[c.Name] = c.Value
	}
	want := map[string]string{"count": "116", "first": "a", "ct": "application/json; charset=utf-8", "code": "200", "tags": `["x","y"]`, "none": "null"}
	if len(got) != len(want) {
		t.Fatalf("captured %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	res := Run(text, r)
	if !res[0].OK || res[0].Detail != "count = 116" || res[6].OK || res[7].OK || res[8].OK {
		t.Fatalf("%+v", res)
	}
	if got := Captures("set a = status", nil); got != nil {
		t.Fatal("nothing to capture without a response")
	}
}
