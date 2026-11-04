package respview

import (
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestTimelineListsOnlyThePhasesThatHappened(t *testing.T) {
	r := &model.Response{Duration: 1200 * time.Millisecond, Size: 2048,
		Timing: model.Timing{Connect: 30 * time.Millisecond, FirstByte: 900 * time.Millisecond, Transfer: 100 * time.Millisecond}}
	got := Timeline(r)
	for _, want := range []string{"Total      1.2s", "Connect    30ms", "First byte 900ms", "Transfer   100ms", "Size       2.00KB"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "DNS") || strings.Contains(got, "TLS") || strings.Contains(got, "cut") {
		t.Errorf("phases that did not happen must not show:\n%s", got)
	}
	r.Truncated = true
	if !strings.Contains(Timeline(r), "is cut here") {
		t.Error("a cut body needs a note")
	}
}
