package app

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var samplingAsked = request.SamplingRequest{
	SystemPrompt: "Be brief",
	Messages:     []request.SamplingMessage{{Role: "user", Text: "Say hi"}},
}

type sampled struct {
	result request.SamplingResult
	ok     bool
}

// ask runs the app's sampler the way a session does, off the UI goroutine.
func ask(app *App, ctx context.Context) chan sampled {
	answer := make(chan sampled, 1)
	go func() {
		result, ok := app.sampleViaDialog(ctx, samplingAsked)
		answer <- sampled{result, ok}
	}()
	return answer
}

func TestTheAppOffersSamplingToMCPServers(t *testing.T) {
	a := testApp(t)
	mcp, ok := a.reg["mcp"].(proto.MCP)
	if !ok || mcp.Sampler == nil {
		t.Fatalf("the mcp protocol is %#v, it must have a Sampler", a.reg["mcp"])
	}
}

func TestASamplingRequestOpensADialogAndTheTypedReplyIsTheAnswer(t *testing.T) {
	a := testApp(t)
	answer := ask(a, context.Background())
	until(t, a, "the sampling dialog", func() bool { return a.dialog != nil })
	if !strings.Contains(a.dialog.Body, "Say hi") || !strings.Contains(a.dialog.Body, "Be brief") {
		t.Fatalf("the dialog must show the request, body = %q", a.dialog.Body)
	}
	a.dialog.Input.Text = "Hello!"
	dialogButtonNamed(t, a, "Send reply").Run()
	got := <-answer
	if !got.ok || got.result.Text != "Hello!" {
		t.Fatalf("answer = %+v", got)
	}
	if a.dialog != nil {
		t.Fatal("the dialog closes after the reply")
	}
}

func TestDecliningASamplingRequestAnswersNo(t *testing.T) {
	a := testApp(t)
	answer := ask(a, context.Background())
	until(t, a, "the sampling dialog", func() bool { return a.dialog != nil })
	dialogButtonNamed(t, a, "Decline").Run()
	if got := <-answer; got.ok {
		t.Fatalf("answer = %+v", got)
	}
}

func TestAnAbandonedSamplingRequestClosesItsDialog(t *testing.T) {
	a := testApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	answer := ask(a, ctx)
	until(t, a, "the sampling dialog", func() bool { return a.dialog != nil })
	cancel()
	if got := <-answer; got.ok {
		t.Fatalf("answer = %+v", got)
	}
	until(t, a, "the dialog to close", func() bool { return a.dialog == nil })
}

func TestASamplingRequestWaitsForTheDialogThatIsOpen(t *testing.T) {
	a := testApp(t)
	a.prompt("Something else", "", "OK", func(string) {})
	answer := ask(a, context.Background())
	for range 20 {
		a.step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}
	if a.dialog.Title != "Something else" {
		t.Fatalf("the sampling request replaced the open dialog: %q", a.dialog.Title)
	}
	a.closeDialog()
	until(t, a, "the sampling dialog after the other closed", func() bool { return a.dialog != nil && a.dialog.Title != "Something else" })
	dialogButtonNamed(t, a, "Decline").Run()
	<-answer
}
