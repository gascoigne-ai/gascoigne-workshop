package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

var samplingAsked = request.SamplingRequest{
	SystemPrompt: "Be brief",
	Messages:     []request.SamplingMessage{{Role: "user", Text: "Say hi"}},
}

type sampled struct {
	result request.SamplingResult
	ok     bool
}

// ask runs the sampler the way a session does, off the UI goroutine.
func ask(world *world, ctx context.Context) chan sampled {
	answer := make(chan sampled, 1)
	go func() {
		result, ok := world.pane.Sample(ctx, samplingAsked)
		answer <- sampled{result, ok}
	}()
	return answer
}

func TestASamplingRequestOpensADialogAndTheTypedReplyIsTheAnswer(t *testing.T) {
	world := newWorld(t)
	answer := ask(world, context.Background())
	world.fake.until(t, "the sampling dialog", func() bool { return world.fake.dialog != nil })
	shown := world.fake.dialog
	if !strings.Contains(shown.state.Body, "Say hi") || !strings.Contains(shown.state.Body, "Be brief") {
		t.Fatalf("the dialog must show the request, body = %q", shown.state.Body)
	}
	shown.state.Input.Text = "Hello!"
	shown.button(t, "Send reply").Run()
	got := <-answer
	if !got.ok || got.result.Text != "Hello!" {
		t.Fatalf("answer = %+v", got)
	}
	if world.fake.dialog != nil {
		t.Fatal("the dialog closes after the reply")
	}
}

func TestDecliningASamplingRequestAnswersNo(t *testing.T) {
	world := newWorld(t)
	answer := ask(world, context.Background())
	world.fake.until(t, "the sampling dialog", func() bool { return world.fake.dialog != nil })
	world.fake.dialog.button(t, "Decline").Run()
	if got := <-answer; got.ok {
		t.Fatalf("answer = %+v", got)
	}
}

func TestAnAbandonedSamplingRequestClosesItsDialog(t *testing.T) {
	world := newWorld(t)
	ctx, cancel := context.WithCancel(context.Background())
	answer := ask(world, ctx)
	world.fake.until(t, "the sampling dialog", func() bool { return world.fake.dialog != nil })
	cancel()
	if got := <-answer; got.ok {
		t.Fatalf("answer = %+v", got)
	}
	world.fake.until(t, "the dialog to close", func() bool { return world.fake.dialog == nil })
}

func TestASamplingRequestWaitsForTheDialogThatIsOpen(t *testing.T) {
	world := newWorld(t)
	world.fake.Open(widgetDialog("Something else"), nil)
	other := world.fake.dialog
	answer := ask(world, context.Background())
	for range 20 {
		world.fake.pump()
		time.Sleep(5 * time.Millisecond)
	}
	if world.fake.dialog != other {
		t.Fatalf("the sampling request replaced the open dialog: %q", world.fake.dialog.state.Title)
	}
	world.fake.dialog = nil
	world.fake.until(t, "the sampling dialog after the other closed", func() bool { return world.fake.dialog != nil })
	world.fake.dialog.button(t, "Decline").Run()
	<-answer
}
