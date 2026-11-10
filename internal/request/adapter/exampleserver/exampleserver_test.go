package exampleserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// start serves every protocol on random ports and returns the endpoint
// URLs by service name. The clients under test are the workshop's own, so
// these tests also prove the example server speaks what they expect.
func start(t *testing.T) map[string]string {
	t.Helper()
	endpoints, err := Listen(Config{Host: "127.0.0.1"})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, endpoints) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	urls := map[string]string{}
	for _, endpoint := range endpoints {
		urls[endpoint.Name] = endpoint.URL
	}
	return urls
}

func within(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func send(t *testing.T, request model.Request) model.Response {
	t.Helper()
	response, err := proto.REST{}.Send(within(t), request)
	if err != nil {
		t.Fatalf("send %s: %v", request.URL, err)
	}
	return response
}

func TestListenOffersEveryProtocolOnItsOwnPort(t *testing.T) {
	urls := start(t)
	for _, name := range []string{"rest", "websocket", "sse", "mcp", "grpc"} {
		if urls[name] == "" {
			t.Errorf("no endpoint for %s in %v", name, urls)
		}
	}
}

func TestRESTEchoReturnsWhatItWasSent(t *testing.T) {
	urls := start(t)
	response := send(t, model.Request{Method: "POST", URL: urls["rest"] + "/echo?x=1", Body: `{"a":1}`, BodyType: model.BodyJSON,
		Headers: []model.KV{{Name: "X-Demo", Value: "yes", Enabled: true}}})
	var echoed struct {
		Method  string
		Query   map[string][]string
		Headers map[string][]string
		Body    string
	}
	if err := json.Unmarshal(response.Body, &echoed); err != nil {
		t.Fatalf("body %q: %v", response.Body, err)
	}
	if echoed.Method != "POST" || echoed.Body != `{"a":1}` || echoed.Query["x"][0] != "1" || echoed.Headers["X-Demo"][0] != "yes" {
		t.Fatalf("echo = %+v", echoed)
	}
}

func TestRESTStatusAnswersWithTheRequestedCode(t *testing.T) {
	urls := start(t)
	tests := []struct {
		path string
		want int
	}{
		{"/status/418", 418},
		{"/status/204", 204},
		{"/status/999", 400},
		{"/status/teapot", 400},
	}
	for _, test := range tests {
		if got := send(t, model.Request{Method: "GET", URL: urls["rest"] + test.path}).Status; got != test.want {
			t.Errorf("GET %s = %d, want %d", test.path, got, test.want)
		}
	}
}

func TestRESTDelayWaitsBeforeAnswering(t *testing.T) {
	urls := start(t)
	began := time.Now()
	response := send(t, model.Request{Method: "GET", URL: urls["rest"] + "/delay/120"})
	if response.Status != 200 || time.Since(began) < 100*time.Millisecond {
		t.Fatalf("status %d after %v, want 200 after about 120ms", response.Status, time.Since(began))
	}
}

func TestRESTAuthNeedsCredentials(t *testing.T) {
	urls := start(t)
	if got := send(t, model.Request{Method: "GET", URL: urls["rest"] + "/auth"}).Status; got != 401 {
		t.Fatalf("without credentials = %d, want 401", got)
	}
	response := send(t, model.Request{Method: "GET", URL: urls["rest"] + "/auth", Auth: model.Auth{Type: model.AuthBearer, Token: "tok"}})
	if response.Status != 200 || !strings.Contains(string(response.Body), "tok") {
		t.Fatalf("with a token = %d %q, want 200 naming the token", response.Status, response.Body)
	}
}

// waitFor reads events until one's text contains want.
func waitFor(t *testing.T, session interface{ Events() <-chan proto.Event }, want string) proto.Event {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event, ok := <-session.Events():
			if !ok {
				t.Fatalf("events ended while waiting for %q", want)
			}
			if strings.Contains(event.Text, want) {
				return event
			}
		case <-timeout:
			t.Fatalf("no event with %q", want)
		}
	}
}

func TestWebSocketEchoesEachMessage(t *testing.T) {
	urls := start(t)
	session, err := proto.WebSocket{}.Connect(within(t), model.Request{Protocol: "websocket", URL: urls["websocket"]})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Send("hello"); err != nil {
		t.Fatal(err)
	}
	if event := waitFor(t, session, "hello"); event.Dir != proto.Sent {
		t.Fatalf("first match is %+v, want the sent message", event)
	}
	if event := waitFor(t, session, "hello"); event.Dir != proto.Received {
		t.Fatalf("second match is %+v, want the echo", event)
	}
}

func TestSSEStreamsTheRequestedNumberOfEvents(t *testing.T) {
	urls := start(t)
	session, err := proto.SSE{}.Connect(within(t), model.Request{Protocol: "sse", URL: urls["sse"] + "?count=3&interval=1ms"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	waitFor(t, session, "tick 3")
}

func TestMCPListsAndCallsTools(t *testing.T) {
	urls := start(t)
	session, err := proto.MCP{}.Connect(within(t), model.Request{Protocol: "mcp", URL: urls["mcp"]})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	waitFor(t, session, "Connected to")
	waitFor(t, session, `"name":"add"`)
	if err := session.Send(`call add {"a":2,"b":3}`); err != nil {
		t.Fatal(err)
	}
	if event := waitFor(t, session, `"text":"5"`); event.Dir != proto.Received {
		t.Fatalf("result %+v, want the received sum", event)
	}
	if err := session.Send(`call echo {"text":"hi"}`); err != nil {
		t.Fatal(err)
	}
	waitFor(t, session, `"text":"hi"`)
}

func TestGRPCServesHealthAndReflection(t *testing.T) {
	urls := start(t)
	request := model.Request{Protocol: "grpc", URL: urls["grpc"], Method: "grpc.health.v1.Health/Check", Body: `{"service": ""}`}
	methods, err := proto.ListGRPCMethods(within(t), request)
	if err != nil {
		t.Fatalf("reflection: %v", err)
	}
	if !strings.Contains(strings.Join(methods, " "), "grpc.health.v1.Health/Check") {
		t.Fatalf("methods = %v, want the health check listed", methods)
	}
	response, err := proto.GRPC{}.Send(within(t), request)
	if err != nil || response.Status != 200 || !strings.Contains(string(response.Body), "SERVING") {
		t.Fatalf("check = %d %q, %v; want SERVING", response.Status, response.Body, err)
	}
}
