// Command example-server serves an example of every protocol the workshop
// speaks, for trying a client without an outside service.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/exampleserver"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "example-server:", err)
		os.Exit(1)
	}
}

func run() error {
	config := exampleserver.DefaultConfig()
	flag.StringVar(&config.Host, "host", config.Host, "address to listen on")
	flag.IntVar(&config.REST, "rest", config.REST, "REST port (0 picks a free one)")
	flag.IntVar(&config.WebSocket, "websocket", config.WebSocket, "WebSocket port")
	flag.IntVar(&config.SSE, "sse", config.SSE, "server-sent events port")
	flag.IntVar(&config.MCP, "mcp", config.MCP, "MCP port")
	flag.IntVar(&config.GRPC, "grpc", config.GRPC, "gRPC port")
	flag.Parse()

	endpoints, err := exampleserver.Listen(config)
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		fmt.Printf("%-10s %s\n", endpoint.Name, endpoint.URL)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return exampleserver.Serve(ctx, endpoints)
}
