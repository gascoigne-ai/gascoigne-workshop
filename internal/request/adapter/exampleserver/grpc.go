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
	server := healthGRPCServer()
	reflection.Register(server)
	return serveUntilDone(ctx, server, listener)
}

// serveGRPCWithoutReflection serves the same health service with no
// reflection, as many production servers do: a client learns the methods
// from protos/grpc/health/v1/health.proto.
func serveGRPCWithoutReflection(ctx context.Context, listener net.Listener) error {
	return serveUntilDone(ctx, healthGRPCServer(), listener)
}

func healthGRPCServer() *grpc.Server {
	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus(exampleHealthService, healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	return server
}

func serveUntilDone(ctx context.Context, server *grpc.Server, listener net.Listener) error {
	defer context.AfterFunc(ctx, server.Stop)()
	return server.Serve(listener)
}
