// Package exampleserver serves a small example of every protocol the
// workshop speaks (REST, WebSocket, SSE, MCP and gRPC), one port each, so a
// client can be tried without an outside service. It is what the debug
// build's sample collection points at.
package exampleserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
)

// Config says where the services listen. A port of 0 picks a free one.
type Config struct {
	Host      string
	REST      int
	WebSocket int
	SSE       int
	MCP       int
	GRPC      int
}

// DefaultConfig is the layout the debug sample collection expects.
func DefaultConfig() Config {
	return Config{Host: "127.0.0.1", REST: 8081, WebSocket: 8082, SSE: 8083, MCP: 8084, GRPC: 8085}
}

// Endpoint is one listening service.
type Endpoint struct {
	Name string // the protocol: rest, websocket, sse, mcp, grpc
	URL  string // what to put in a request to reach it

	listener net.Listener
	serve    func(ctx context.Context, listener net.Listener) error
}

// service is one row of the table of protocols: adding a protocol means
// adding a row, not a branch.
type service struct {
	name  string
	port  func(config Config) int
	url   func(address string) string
	serve func(ctx context.Context, listener net.Listener) error
}

var services = []service{
	{"rest", func(config Config) int { return config.REST }, func(address string) string { return "http://" + address }, serveHTTP(restHandler())},
	{"websocket", func(config Config) int { return config.WebSocket }, func(address string) string { return "ws://" + address }, serveHTTP(http.HandlerFunc(echoWebSocket))},
	{"sse", func(config Config) int { return config.SSE }, func(address string) string { return "http://" + address + "/events" }, serveHTTP(sseHandler())},
	{"mcp", func(config Config) int { return config.MCP }, func(address string) string { return "http://" + address + "/mcp" }, serveHTTP(mcpHandler())},
	{"grpc", func(config Config) int { return config.GRPC }, func(address string) string { return "grpc://" + address }, serveGRPC},
}

// Listen opens the port of every service. It opens all or none.
func Listen(config Config) ([]*Endpoint, error) {
	host := config.Host
	if host == "" {
		host = "127.0.0.1"
	}
	var endpoints []*Endpoint
	for _, service := range services {
		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(service.port(config))))
		if err != nil {
			closeAll(endpoints)
			return nil, fmt.Errorf("listen for %s: %w", service.name, err)
		}
		endpoints = append(endpoints, &Endpoint{
			Name: service.name, URL: service.url(listener.Addr().String()),
			listener: listener, serve: service.serve,
		})
	}
	return endpoints, nil
}

func closeAll(endpoints []*Endpoint) {
	for _, endpoint := range endpoints {
		endpoint.listener.Close()
	}
}

// Serve runs every endpoint until ctx ends or one fails; then it stops all
// of them and returns the first failure, nil after a plain shutdown.
func Serve(ctx context.Context, endpoints []*Endpoint) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	results := make(chan error, len(endpoints))
	for _, endpoint := range endpoints {
		go func() {
			err := endpoint.serve(ctx, endpoint.listener)
			if err != nil {
				err = fmt.Errorf("serve %s: %w", endpoint.Name, err)
			}
			results <- err
		}()
	}
	var first error
	for range endpoints {
		if err := <-results; err != nil && first == nil {
			first = err
		}
		stop() // one service ending, for any reason, ends the others
	}
	return first
}

// serveHTTP makes a serve function for an HTTP handler.
func serveHTTP(handler http.Handler) func(ctx context.Context, listener net.Listener) error {
	return func(ctx context.Context, listener net.Listener) error {
		server := &http.Server{Handler: handler}
		defer context.AfterFunc(ctx, func() { server.Close() })()
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
