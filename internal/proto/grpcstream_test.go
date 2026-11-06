package proto

import (
	"strings"
	"testing"
	"time"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// nextEvent waits for the next event of session or fails.
func nextEvent(t *testing.T, session Session) Event {
	t.Helper()
	select {
	case event, ok := <-session.Events():
		if !ok {
			t.Fatal("the session ended")
		}
		return event
	case <-time.After(5 * time.Second):
		t.Fatal("no event in 5 seconds")
	}
	return Event{}
}

// nextReceived skips events until a message from the server arrives.
func nextReceived(t *testing.T, session Session) Event {
	t.Helper()
	for {
		if event := nextEvent(t, session); event.Dir == Received {
			return event
		}
	}
}

func TestGRPCServerStreamShowsEveryMessageAsAnEvent(t *testing.T) {
	target, _, healthServer := grpcStubWithHealth(t)
	session, err := GRPC{}.Connect(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Watch", `{"service": "known"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	if opened := nextEvent(t, session); opened.Dir != Info || opened.Kind != "open" {
		t.Fatalf("first event = %+v, want the open note", opened)
	}
	if first := nextReceived(t, session); !strings.Contains(first.Text, `"SERVING"`) {
		t.Fatalf("first message = %q", first.Text)
	}
	healthServer.SetServingStatus("known", healthpb.HealthCheckResponse_NOT_SERVING)
	if second := nextReceived(t, session); !strings.Contains(second.Text, `"NOT_SERVING"`) {
		t.Fatalf("second message = %q", second.Text)
	}
}

func TestGRPCClosingTheSessionEndsTheStream(t *testing.T) {
	target, _ := grpcStub(t)
	session, err := GRPC{}.Connect(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Watch", `{"service": "known"}`))
	if err != nil {
		t.Fatal(err)
	}
	nextReceived(t, session)
	session.Close()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-session.Events():
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("the event channel stayed open after Close")
		}
	}
}

func TestGRPCBidirectionalStreamSendsComposedMessages(t *testing.T) {
	target, _ := grpcStub(t)
	session, err := GRPC{}.Connect(grpcContext(t), grpcRequest(target, "grpc.reflection.v1.ServerReflection/ServerReflectionInfo", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Send(`{"listServices": ""}`); err != nil {
		t.Fatal(err)
	}
	var sent bool
	for {
		event := nextEvent(t, session)
		sent = sent || (event.Dir == Sent && strings.Contains(event.Text, "listServices"))
		if event.Dir == Received {
			if !sent || !strings.Contains(event.Text, "grpc.health.v1.Health") {
				t.Fatalf("sent=%v, answer %q", sent, event.Text)
			}
			return
		}
	}
}

func TestGRPCSendingTextThatIsNotAMessageIsAnError(t *testing.T) {
	target, _ := grpcStub(t)
	session, err := GRPC{}.Connect(grpcContext(t), grpcRequest(target, "grpc.reflection.v1.ServerReflection/ServerReflectionInfo", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Send("not json"); err == nil {
		t.Fatal("a message that is not JSON was accepted")
	}
}

func TestGRPCUnaryMethodsAreRefusedByConnect(t *testing.T) {
	target, _ := grpcStub(t)
	_, err := GRPC{}.Connect(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Check", "{}"))
	if err == nil || !strings.Contains(err.Error(), "unary") {
		t.Fatalf("err = %v, want a note that the method is unary", err)
	}
}

func TestGRPCStreamingMethodsAreStillRefusedByTheOneShotSend(t *testing.T) {
	target, _ := grpcStub(t)
	response, _ := GRPC{}.Send(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Watch", "{}"))
	if !strings.Contains(response.Err, "streaming") {
		t.Fatalf("err = %q", response.Err)
	}
}
