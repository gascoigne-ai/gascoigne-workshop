package ui

import (
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// grpcServer runs a real gRPC server with reflection and the standard
// health service (see the proto package's tests for why).
func grpcServer(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	healthpb.RegisterHealthServer(server, health.NewServer())
	reflection.Register(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return "grpc://" + listener.Addr().String()
}

func TestTheProtocolChipCyclesThroughGRPCAndBackToRESTWithAMethod(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	seen := map[string]bool{}
	for range protocolOrder {
		a.cycleProtocol(tab)
		seen[tab.req.Protocol] = true
		if tab.req.Protocol == "grpc" {
			tab.req.Method = "pkg.Service/Call" // a gRPC method is not an HTTP one
		}
	}
	if !seen["grpc"] {
		t.Fatalf("the protocols are %v, gRPC is missing", protocolOrder)
	}
	if tab.req.Protocol != "rest" || tab.req.Method != "GET" {
		t.Fatalf("back on %s %q: a REST request needs an HTTP method", tab.req.Protocol, tab.req.Method)
	}
}

func TestPickingAGRPCMethodListsTheServersMethodsAndSetsIt(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol, tab.req.URL = "grpc", grpcServer(t)

	a.pickGRPCMethod(tab)
	for range 400 {
		a.step(core.Input{})
		if a.ctxMenu != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if a.ctxMenu == nil {
		t.Fatal("no menu of methods opened")
	}
	if !strings.Contains(strings.Join(menuLabels(a), " "), "grpc.health.v1.Health/Check") {
		t.Fatalf("methods = %v", menuLabels(a))
	}
	itemByLabel(a, "grpc.health.v1.Health/Check").run()
	if tab.req.Method != "grpc.health.v1.Health/Check" {
		t.Fatalf("Method = %q", tab.req.Method)
	}
}

func TestSendingAGRPCTabShowsTheAnswerAsAResponse(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol, tab.req.URL, tab.req.Method, tab.req.Body = "grpc", grpcServer(t), "grpc.health.v1.Health/Check", `{}`
	tab.resp = nil
	a.send(tab)
	waitForResponse(t, a, tab)
	if tab.resp.Err != "" || tab.resp.Status != 200 || !strings.Contains(string(tab.resp.Body), "SERVING") {
		t.Fatalf("%+v", tab.resp)
	}
}

func TestAGRPCPickerWithoutAServerSaysSo(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol, tab.req.URL = "grpc", ""
	a.pickGRPCMethod(tab)
	a.step(core.Input{})
	if a.ctxMenu != nil {
		t.Fatal("a menu opened without a server")
	}
	if len(a.toasts) == 0 {
		t.Fatal("no message tells the user a server is needed")
	}
}
