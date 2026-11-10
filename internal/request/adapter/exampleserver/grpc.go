package exampleserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// exampleHealthService is a service name that reports SERVING besides the
// server as a whole (the empty name).
const exampleHealthService = "example"

// serveGRPC serves the standard health service, with reflection so the
// workshop can list its methods.
func serveGRPC(ctx context.Context, listener net.Listener) error {
	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus(exampleHealthService, healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	reflection.Register(server)
	defer context.AfterFunc(ctx, server.Stop)()
	return server.Serve(listener)
}
