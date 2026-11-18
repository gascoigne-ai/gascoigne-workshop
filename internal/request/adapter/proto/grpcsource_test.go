package proto

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// grpcStubWithoutReflection serves the standard health service and no
// reflection, so only a .proto file can tell what it offers.
func grpcStubWithoutReflection(t *testing.T) (target string, healthServer *health.Server) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	healthServer = health.NewServer()
	healthServer.SetServingStatus("known", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return "grpc://" + listener.Addr().String(), healthServer
}

func fromHealthProto(t *testing.T, request model.Request) model.Request {
	t.Helper()
	request.Proto = model.ProtoSource{Files: []string{absolute(t, healthProto)}}
	return request
}

func TestGRPCUnaryCallWorksFromAProtoFileWithoutReflection(t *testing.T) {
	target, _ := grpcStubWithoutReflection(t)
	request := grpcRequest(target, "grpc.health.v1.Health/Check", `{"service": "known"}`)

	// The same request without the file has nothing to learn the method from.
	withoutFile, err := GRPC{}.Send(grpcContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if withoutFile.Err == "" {
		t.Fatalf("a server without reflection answered without a proto file: %+v", withoutFile)
	}

	response, err := GRPC{}.Send(grpcContext(t), fromHealthProto(t, request))
	if err != nil {
		t.Fatal(err)
	}
	if response.Err != "" || response.Status != 200 {
		t.Fatalf("status %d err %q", response.Status, response.Err)
	}
	if want := "{\n  \"status\": \"SERVING\"\n}"; string(response.Body) != want {
		t.Fatalf("body = %q", response.Body)
	}
}

func TestListGRPCMethodsReadsTheProtoFiles(t *testing.T) {
	target, _ := grpcStubWithoutReflection(t)
	methods, err := ListGRPCMethods(grpcContext(t), fromHealthProto(t, grpcRequest(target, "", "")))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"grpc.health.v1.Health/Check", "grpc.health.v1.Health/List", "grpc.health.v1.Health/Watch"}
	if !reflect.DeepEqual(methods, want) {
		t.Errorf("methods = %v, want %v", methods, want)
	}
}

// The Proto tab lists methods and checks the files before a server is
// entered or reachable: the files alone say what there is.
func TestListGRPCMethodsFromProtoFilesNeedsNoServer(t *testing.T) {
	for _, url := range []string{"", "grpc://127.0.0.1:1"} {
		methods, err := ListGRPCMethods(grpcContext(t), fromHealthProto(t, grpcRequest(url, "", "")))
		if err != nil {
			t.Fatalf("URL %q: %v", url, err)
		}
		if len(methods) != 3 {
			t.Errorf("URL %q: methods = %v, want the three of Health", url, methods)
		}
	}
	if _, err := ListGRPCMethods(grpcContext(t), grpcRequest("", "", "")); err == nil {
		t.Error("reflection without a server listed methods")
	}
}

func TestGRPCServerStreamWorksFromAProtoFile(t *testing.T) {
	target, healthServer := grpcStubWithoutReflection(t)
	request := fromHealthProto(t, grpcRequest(target, "grpc.health.v1.Health/Watch", `{"service": "known"}`))
	session, err := GRPC{}.Connect(grpcContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	if first := nextReceived(t, session); !strings.Contains(first.Text, `"SERVING"`) {
		t.Fatalf("first message = %q", first.Text)
	}
	healthServer.SetServingStatus("known", healthpb.HealthCheckResponse_NOT_SERVING)
	if second := nextReceived(t, session); !strings.Contains(second.Text, `"NOT_SERVING"`) {
		t.Fatalf("second message = %q", second.Text)
	}
}

func TestGRPCAMethodTheProtoFilesLackIsAnErrorResponse(t *testing.T) {
	target, _ := grpcStubWithoutReflection(t)
	request := fromHealthProto(t, grpcRequest(target, "no.such.Service/Nothing", `{}`))
	response, err := GRPC{}.Send(grpcContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Err, "no.such.Service") {
		t.Fatalf("Err = %q", response.Err)
	}
}

func TestGRPCAProtoFileThatIsGoneAtSendIsAnErrorResponseNamingIt(t *testing.T) {
	// The file was chosen, then deleted or moved: the send says which
	// file is missing instead of falling back to reflection.
	target, _ := grpcStub(t)
	gone := filepath.Join(t.TempDir(), "health.proto")
	request := grpcRequest(target, "grpc.health.v1.Health/Check", `{}`)
	request.Proto = model.ProtoSource{Files: []string{gone}}
	response, err := GRPC{}.Send(grpcContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Err, gone) {
		t.Fatalf("Err = %q, want it to name %s", response.Err, gone)
	}
}

func TestGRPCAParseErrorAtSendNamesTheFileAndLine(t *testing.T) {
	source, err := os.ReadFile(healthProto)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(source), "\n")
	broken := filepath.Join(t.TempDir(), "health.proto")
	if err := os.WriteFile(broken, []byte(strings.Join(lines[:33], "")), 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := grpcStubWithoutReflection(t)
	request := grpcRequest(target, "grpc.health.v1.Health/Check", `{}`)
	request.Proto = model.ProtoSource{Files: []string{broken}}
	_, err = openMethodSource(grpcContext(t), nil, request.Proto)
	var protoErr model.ProtoError
	if !errors.As(err, &protoErr) || protoErr.Path != broken || protoErr.Line != 34 {
		t.Fatalf("err = %v, want a ProtoError at %s:34", err, broken)
	}
	response, err := GRPC{}.Send(grpcContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Err, broken+":34:") {
		t.Fatalf("Err = %q, want it to point at %s:34", response.Err, broken)
	}
}
