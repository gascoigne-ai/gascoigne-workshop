//go:build !release

package exampleserver

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// The debug sample collection hard-codes the example server's default
// ports; this keeps the two in step. A gRPC request that brings its own
// .proto files is for the port without reflection.
func TestDebugSampleRequestsUseTheDefaultPorts(t *testing.T) {
	defaults := DefaultConfig()
	for _, call := range collection.Sample().Requests {
		service := sampleService(*call)
		parsed, err := url.Parse(call.URL)
		if err != nil {
			t.Fatalf("%s: %v", call.Name, err)
		}
		_, port, err := net.SplitHostPort(parsed.Host)
		if want := portOf(t, service, defaults); err != nil || port != strconv.Itoa(want) {
			t.Errorf("%s (%s) uses %q, want port %d", call.Name, service, parsed.Host, want)
		}
	}
}

// Every service the example server runs is reached by a sample request,
// so a person finds each one in the tree.
func TestDebugSampleHasARequestForEveryService(t *testing.T) {
	reached := map[string]bool{}
	for _, call := range collection.Sample().Requests {
		reached[sampleService(*call)] = true
	}
	for _, service := range services {
		if !reached[service.name] {
			t.Errorf("no sample request for the %s service", service.name)
		}
	}
}

// The sample's proto file is a real one that the app can read, wherever
// the program was started from.
func TestDebugSampleProtoFilesExistAndAreAbsolute(t *testing.T) {
	for _, call := range collection.Sample().Requests {
		for _, file := range call.Proto.Files {
			if !filepath.IsAbs(file) {
				t.Errorf("%s names %q, which depends on the working folder", call.Name, file)
			}
			if _, err := os.Stat(file); err != nil {
				t.Errorf("%s: %v", call.Name, err)
			}
		}
	}
}

// sampleService is the example server's service a sample request is for.
func sampleService(call request.Request) string {
	if call.Protocol == "grpc" && call.Proto.Kind() == request.SourceProtoFiles {
		return "grpc-no-reflection"
	}
	return string(call.Protocol)
}

func portOf(t *testing.T, name string, config Config) int {
	t.Helper()
	for _, service := range services {
		if service.name == name {
			return service.port(config)
		}
	}
	t.Fatalf("no service %q", name)
	return 0
}

// matchesPattern reports whether path is served by a route pattern such
// as "/status/{code}", where a {name} stands for one segment.
func matchesPattern(pattern, path string) bool {
	if pattern == "/{$}" {
		pattern = "/"
	}
	want, have := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(want) != len(have) {
		return false
	}
	for i, part := range want {
		if !strings.HasPrefix(part, "{") && part != have[i] {
			return false
		}
	}
	return true
}

// The sample collection is how a person finds the example server's
// endpoints, so each route of the REST example needs a request.
func TestDebugSampleHasARequestForEveryRESTRoute(t *testing.T) {
	sample := collection.Sample()
	for _, route := range restRoutes() {
		found := false
		for _, call := range sample.Requests {
			parsed, err := url.Parse(call.URL)
			if err == nil && call.Protocol == "rest" && matchesPattern(route.pattern, parsed.Path) {
				found = true
			}
		}
		if !found {
			t.Errorf("no sample request for %s", route.pattern)
		}
	}
}
