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

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// The debug sample collection hard-codes the example server's default
// ports; this keeps the two in step. A gRPC request that brings its own
// .proto files is for the port without reflection.
func TestDebugSampleRequestsUseTheDefaultPorts(t *testing.T) {
	defaults := DefaultConfig()
	for _, request := range collection.Sample().Requests {
		service := sampleService(*request)
		parsed, err := url.Parse(request.URL)
		if err != nil {
			t.Fatalf("%s: %v", request.Name, err)
		}
		_, port, err := net.SplitHostPort(parsed.Host)
		if want := portOf(t, service, defaults); err != nil || port != strconv.Itoa(want) {
			t.Errorf("%s (%s) uses %q, want port %d", request.Name, service, parsed.Host, want)
		}
	}
}

// Every service the example server runs is reached by a sample request,
// so a person finds each one in the tree.
func TestDebugSampleHasARequestForEveryService(t *testing.T) {
	reached := map[string]bool{}
	for _, request := range collection.Sample().Requests {
		reached[sampleService(*request)] = true
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
	for _, request := range collection.Sample().Requests {
		for _, file := range request.Proto.Files {
			if !filepath.IsAbs(file) {
				t.Errorf("%s names %q, which depends on the working folder", request.Name, file)
			}
			if _, err := os.Stat(file); err != nil {
				t.Errorf("%s: %v", request.Name, err)
			}
		}
	}
}

// sampleService is the example server's service a sample request is for.
func sampleService(request model.Request) string {
	if request.Protocol == "grpc" && request.Proto.Kind() == model.SourceProtoFiles {
		return "grpc-no-reflection"
	}
	return request.Protocol
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
		for _, request := range sample.Requests {
			parsed, err := url.Parse(request.URL)
			if err == nil && request.Protocol == "rest" && matchesPattern(route.pattern, parsed.Path) {
				found = true
			}
		}
		if !found {
			t.Errorf("no sample request for %s", route.pattern)
		}
	}
}
