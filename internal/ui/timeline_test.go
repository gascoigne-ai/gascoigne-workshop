package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestTimelineListsOnlyThePhasesThatHappened(t *testing.T) {
	r := &model.Response{Duration: 1200 * time.Millisecond, Size: 2048,
		Timing: model.Timing{Connect: 30 * time.Millisecond, FirstByte: 900 * time.Millisecond, Transfer: 100 * time.Millisecond}}
	got := timelineText(r)
	for _, want := range []string{"Total      1.2s", "Connect    30ms", "First byte 900ms", "Transfer   100ms", "Size       2.00KB"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "DNS") || strings.Contains(got, "TLS") || strings.Contains(got, "cut") {
		t.Errorf("phases that did not happen must not show:\n%s", got)
	}
	r.Truncated = true
	if !strings.Contains(timelineText(r), "is cut here") {
		t.Error("a cut body needs a note")
	}
}

func TestRequestsShareACookieJarThatClearCookiesEmpties(t *testing.T) {
	a, _ := newTestApp(t)
	first := a.jar
	if first == nil {
		t.Fatal("no cookie jar")
	}
	rest, ok := a.reg["rest"].(proto.REST)
	if !ok || rest.Jar != first {
		t.Fatalf("the REST protocol must use the app's jar: %+v", a.reg["rest"])
	}
	a.clearCookies()
	rest, _ = a.reg["rest"].(proto.REST)
	if a.jar == first || rest.Jar != a.jar {
		t.Fatal("clearing cookies must install a new jar everywhere")
	}
}
