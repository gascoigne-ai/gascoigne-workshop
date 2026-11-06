package proto

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	reflectionv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// grpcStub runs a real gRPC server with reflection and the standard
// health service, which ships with grpc and so needs no invented
// protobuf fixtures. seen receives the metadata of every Check call.
func grpcStub(t *testing.T) (target string, seen chan metadata.MD) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	seen = make(chan metadata.MD, 4)
	server := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, request any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if incoming, ok := metadata.FromIncomingContext(ctx); ok {
			seen <- incoming
		}
		return handler(ctx, request)
	}))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("known", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	reflection.Register(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return "grpc://" + listener.Addr().String(), seen
}

func grpcContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func grpcRequest(target, method, body string) model.Request {
	return model.Request{Protocol: "grpc", URL: target, Method: method, Body: body}
}

func TestListGRPCMethodsReadsTheServersReflection(t *testing.T) {
	target, _ := grpcStub(t)
	methods, err := ListGRPCMethods(grpcContext(t), grpcRequest(target, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, method := range methods {
		found = found || method == "grpc.health.v1.Health/Check"
		if strings.HasPrefix(method, "grpc.reflection.") {
			t.Errorf("the reflection service itself is listed: %s", method)
		}
	}
	if !found {
		t.Fatalf("grpc.health.v1.Health/Check missing from %v", methods)
	}
}

func TestGRPCUnaryCallTakesJSONAndAnswersJSON(t *testing.T) {
	target, _ := grpcStub(t)
	response, err := GRPC{}.Send(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Check", `{"service": "known"}`))
	if err != nil {
		t.Fatal(err)
	}
	if response.Err != "" || response.Status != 200 || response.StatusText != "OK" {
		t.Fatalf("status %d %q err %q", response.Status, response.StatusText, response.Err)
	}
	if want := "{\n  \"status\": \"SERVING\"\n}"; string(response.Body) != want {
		t.Fatalf("body = %q", response.Body)
	}
}

func TestGRPCFailureMapsTheCodeToAStatusLine(t *testing.T) {
	target, _ := grpcStub(t)
	response, err := GRPC{}.Send(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Check", `{"service": "nobody"}`))
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != 404 || response.StatusText != codes.NotFound.String() {
		t.Fatalf("status %d %q", response.Status, response.StatusText)
	}
	if !strings.Contains(string(response.Body), "unknown service") {
		t.Fatalf("the server's message is the body: %q", response.Body)
	}
}

func TestGRPCUnknownMethodIsAnErrorResponseNotACrash(t *testing.T) {
	target, _ := grpcStub(t)
	response, err := GRPC{}.Send(grpcContext(t), grpcRequest(target, "no.such.Service/Nothing", `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if response.Err == "" || !strings.Contains(response.Err, "no.such.Service") {
		t.Fatalf("Err = %q", response.Err)
	}
}

func TestGRPCBadJSONInputIsAnErrorResponse(t *testing.T) {
	target, _ := grpcStub(t)
	response, err := GRPC{}.Send(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Check", `{"nope": 1}`))
	if err != nil {
		t.Fatal(err)
	}
	if response.Err == "" {
		t.Fatalf("a field the message does not have must be refused: %+v", response)
	}
}

func TestGRPCSendsHeadersAsMetadata(t *testing.T) {
	target, seen := grpcStub(t)
	request := grpcRequest(target, "grpc.health.v1.Health/Check", `{}`)
	request.Headers = []model.KV{{Name: "X-Trace", Value: "abc", Enabled: true}, {Name: "X-Off", Value: "no"}}
	if _, err := (GRPC{}).Send(grpcContext(t), request); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-seen:
		if v := got.Get("x-trace"); len(v) != 1 || v[0] != "abc" {
			t.Errorf("x-trace = %v", v)
		}
		if len(got.Get("x-off")) != 0 {
			t.Error("a disabled header was sent")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the server saw no call")
	}
}

func TestGRPCNothingListeningIsAnErrorResponse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	target := "grpc://" + listener.Addr().String()
	_ = listener.Close()
	request := grpcRequest(target, "grpc.health.v1.Health/Check", `{}`)
	response, err := GRPC{}.Send(grpcContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Err == "" {
		t.Fatalf("no error: %+v", response)
	}
}

// Many servers in the wild only have the older v1alpha reflection service.
func TestGRPCFallsBackToTheV1AlphaReflectionService(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	healthpb.RegisterHealthServer(server, health.NewServer())
	reflectionv1alpha.RegisterServerReflectionServer(server, reflection.NewServer(reflection.ServerOptions{Services: server}))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	response, err := GRPC{}.Send(grpcContext(t), grpcRequest("grpc://"+listener.Addr().String(), "grpc.health.v1.Health/Check", `{}`))
	if err != nil || response.Err != "" || response.Status != 200 {
		t.Fatalf("%+v %v", response, err)
	}
}
