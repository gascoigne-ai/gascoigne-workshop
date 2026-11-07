package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var samplingAsked = proto.SamplingRequest{
	SystemPrompt: "Be brief",
	Messages:     []proto.SamplingMessage{{Role: "user", Text: "Say hi"}},
}

type sampled struct {
	result proto.SamplingResult
	ok     bool
}

// ask runs the app's sampler the way a session does, off the UI goroutine.
func ask(a *App, ctx context.Context) chan sampled {
	answer := make(chan sampled, 1)
	go func() {
		result, ok := a.sampleViaDialog(ctx, samplingAsked)
		answer <- sampled{result, ok}
	}()
	return answer
}

// stepUntil runs frames until cond holds.
func stepUntil(t *testing.T, a *App, what string, cond func() bool) {
	t.Helper()
	for range 400 {
		a.step(core.Input{})
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestTheAppOffersSamplingToMCPServers(t *testing.T) {
	a, _ := newTestApp(t)
	mcp, ok := a.reg["mcp"].(proto.MCP)
	if !ok || mcp.Sampler == nil {
		t.Fatalf("the mcp protocol is %#v, it must have a Sampler", a.reg["mcp"])
	}
}

func TestASamplingRequestOpensADialogAndTheTypedReplyIsTheAnswer(t *testing.T) {
	a, _ := newTestApp(t)
	answer := ask(a, context.Background())
	stepUntil(t, a, "the sampling dialog", func() bool { return a.dialog != nil })
	if !strings.Contains(a.dialog.body, "Say hi") || !strings.Contains(a.dialog.body, "Be brief") {
		t.Fatalf("the dialog must show the request, body = %q", a.dialog.body)
	}
	a.dialog.input.Text = "Hello!"
	dialogButtonNamed(t, a, "Send reply").run()
	got := <-answer
	if !got.ok || got.result.Text != "Hello!" {
		t.Fatalf("answer = %+v", got)
	}
	if a.dialog != nil {
		t.Fatal("the dialog closes after the reply")
	}
}

func TestDecliningASamplingRequestAnswersNo(t *testing.T) {
	a, _ := newTestApp(t)
	answer := ask(a, context.Background())
	stepUntil(t, a, "the sampling dialog", func() bool { return a.dialog != nil })
	dialogButtonNamed(t, a, "Decline").run()
	if got := <-answer; got.ok {
		t.Fatalf("answer = %+v", got)
	}
}

func TestAnAbandonedSamplingRequestClosesItsDialog(t *testing.T) {
	a, _ := newTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	answer := ask(a, ctx)
	stepUntil(t, a, "the sampling dialog", func() bool { return a.dialog != nil })
	cancel()
	if got := <-answer; got.ok {
		t.Fatalf("answer = %+v", got)
	}
	stepUntil(t, a, "the dialog to close", func() bool { return a.dialog == nil })
}

func TestASamplingRequestWaitsForTheDialogThatIsOpen(t *testing.T) {
	a, _ := newTestApp(t)
	a.prompt("Something else", "", "OK", func(string) {})
	answer := ask(a, context.Background())
	for range 20 {
		a.step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}
	if a.dialog.title != "Something else" {
		t.Fatalf("the sampling request replaced the open dialog: %q", a.dialog.title)
	}
	a.closeDialog()
	stepUntil(t, a, "the sampling dialog after the other closed", func() bool { return a.dialog != nil && a.dialog.title != "Something else" })
	dialogButtonNamed(t, a, "Decline").run()
	<-answer
}
