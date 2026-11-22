package assert

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// Pins how a header repeated in a response is read before the lookup is
// shared: a check sees every value joined, a capture takes the first.
func TestRepeatedHeaderIsJoinedForChecksAndFirstForCaptures(t *testing.T) {
	r := &request.Response{Status: 200, Headers: []request.KV{{Name: "Set-Cookie", Value: "a=1"}, {Name: "set-cookie", Value: "b=2"}}}
	if res := Run(`header Set-Cookie == "a=1, b=2"`, r); !res[0].OK {
		t.Errorf("joined check: %+v", res[0])
	}
	if res := Run("header Set-Cookie != nope\nheader Nope == x", r); !res[0].OK || res[1].OK || res[1].Detail != "the header is not there" {
		t.Errorf("%+v", res)
	}
	caps := Captures("set c = header set-cookie", r)
	if len(caps) != 1 || caps[0].Value != "a=1" {
		t.Errorf("captures %+v", caps)
	}
}

func TestNumberComparisonsReportTheSameWayForStatusAndJSON(t *testing.T) {
	for line, detail := range map[string]string{
		"status >= 201":             "got 200",
		"json total_count >= 117":   "got 116",
		"json total_count > banana": `"banana" is not a number`,
		"json total_count 5":        "expected exists, contains or an operator",
		"json total_count <= 116":   "",
		"json total_count != 116":   "got 116",
		"status ==200":              "",
	} {
		res := Run(line, resp())
		if (detail == "") != res[0].OK || !strings.Contains(res[0].Detail, detail) {
			t.Errorf("%q: ok=%v detail=%q, want %q", line, res[0].OK, res[0].Detail, detail)
		}
	}
}
