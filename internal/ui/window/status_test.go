package window

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The status pill over the response pane, as the window shows it: nothing
// without an answer or with an error, else the code, its text, the time and
// the size. (The colors of the pill are pinned in ui/response.)
func TestTheWindowShowsTheStatusPillOfTheAnswerOnly(t *testing.T) {
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
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			a.cur().Resp = tc.resp
			dump := Dump(a.view())
			if tc.want == nil {
				for _, soft := range []string{"bg=ok-soft", "bg=warn-soft", "bg=error-soft"} {
					if strings.Contains(dump, soft) {
						t.Fatalf("a pill (%s) is shown for %s:\n%s", soft, tc.name, dump)
					}
				}
				return
			}
			for _, want := range tc.want {
				if !strings.Contains(dump, want) {
					t.Fatalf("the window lacks %q:\n%s", want, dump)
				}
			}
		})
	}
}
