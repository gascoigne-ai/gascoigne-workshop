package proto

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// The fixtures are real files: health.proto and binarylog.proto from
// grpc/grpc-proto, proto2.proto and proto2_ext2.proto from grpc-go's
// reflection tests, each kept at its own path below testdata/protos so
// the imports between them read as upstream wrote them.
const (
	protoRoot       = "testdata/protos"
	healthProto     = "testdata/protos/grpc/health/v1/health.proto"
	binarylogProto  = "testdata/protos/grpc/binlog/v1/binarylog.proto"
	proto2ExtProto  = "testdata/protos/reflection/grpc_testing/proto2_ext2.proto"
	healthService   = "grpc.health.v1.Health"
	healthCheckName = "grpc.health.v1.HealthCheckRequest"
)

func absolute(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestProtoFilesGiveTheServicesTheyDeclare(t *testing.T) {
	files, err := compileProtoFiles(grpcContext(t), "", []string{absolute(t, healthProto)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	service := findCompiledService(t, files, healthService)
	var methods []string
	for i := 0; i < service.Methods().Len(); i++ {
		methods = append(methods, string(service.Methods().Get(i).Name()))
	}
	if got, want := strings.Join(methods, " "), "Check List Watch"; got != want {
		t.Errorf("methods = %q, want %q", got, want)
	}
	if !service.Methods().ByName("Watch").IsStreamingServer() {
		t.Error("Watch is not a server stream")
	}
}

func TestProtoFilesDescribeTheMessages(t *testing.T) {
	files, err := compileProtoFiles(grpcContext(t), "", []string{absolute(t, healthProto)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	found, err := files.AsResolver().FindDescriptorByName(healthCheckName)
	if err != nil {
		t.Fatal(err)
	}
	message, ok := found.(protoreflect.MessageDescriptor)
	if !ok {
		t.Fatalf("%s is a %T, not a message", healthCheckName, found)
	}
	field := message.Fields().ByName("service")
	if field == nil || field.Kind() != protoreflect.StringKind || field.Number() != 1 {
		t.Errorf("field service = %v, want string 1", field)
	}
}

func TestRelativeProtoPathsAreResolvedAgainstTheBaseDirectory(t *testing.T) {
	base := absolute(t, protoRoot)
	files, err := compileProtoFiles(grpcContext(t), base, []string{"grpc/health/v1/health.proto"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	findCompiledService(t, files, healthService)
}

func TestWellKnownImportsNeedNoImportPath(t *testing.T) {
	files, err := compileProtoFiles(grpcContext(t), "", []string{absolute(t, binarylogProto)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	found, err := files.AsResolver().FindDescriptorByName("grpc.binarylog.v1.GrpcLogEntry")
	if err != nil {
		t.Fatal(err)
	}
	timestamp := found.(protoreflect.MessageDescriptor).Fields().ByName("timestamp")
	if got := timestamp.Message().FullName(); got != "google.protobuf.Timestamp" {
		t.Errorf("timestamp is a %s, want google.protobuf.Timestamp", got)
	}
}

func TestImportsAreFoundOnTheImportPath(t *testing.T) {
	files, err := compileProtoFiles(grpcContext(t), "", []string{absolute(t, proto2ExtProto)}, []string{absolute(t, protoRoot)})
	if err != nil {
		t.Fatal(err)
	}
	// frob extends ToBeExtended, which only the imported file declares.
	found, err := files.AsResolver().FindDescriptorByName("grpc.testing.frob")
	if err != nil {
		t.Fatal(err)
	}
	extended := found.(protoreflect.ExtensionDescriptor).ContainingMessage()
	if got := extended.FullName(); got != "grpc.testing.ToBeExtended" {
		t.Errorf("frob extends %s, want grpc.testing.ToBeExtended", got)
	}
}

func TestImportPathsAreSearchedInOrder(t *testing.T) {
	// The first import path holds a copy of the imported proto2.proto cut
	// after line 21, inside "message ToBeExtended {", the second the whole
	// file: the error in the copy shows that the first path was read.
	source, err := os.ReadFile(filepath.Join(protoRoot, "reflection/grpc_testing/proto2.proto"))
	if err != nil {
		t.Fatal(err)
	}
	first := t.TempDir()
	copied := filepath.Join(first, "reflection", "grpc_testing", "proto2.proto")
	if err := os.MkdirAll(filepath.Dir(copied), 0o700); err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(source), "\n")
	if err := os.WriteFile(copied, []byte(strings.Join(lines[:21], "")), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = compileProtoFiles(grpcContext(t), "", []string{absolute(t, proto2ExtProto)}, []string{first, absolute(t, protoRoot)})
	var protoErr model.ProtoError
	if !errors.As(err, &protoErr) || protoErr.Path != copied {
		t.Errorf("err = %v, want one in %s", err, copied)
	}
}

func TestAMissingImportIsAnErrorAtTheImportLine(t *testing.T) {
	// proto2_ext2.proto imports reflection/grpc_testing/proto2.proto,
	// which only the testdata root holds; without it the import fails.
	path := absolute(t, proto2ExtProto)
	_, err := compileProtoFiles(grpcContext(t), "", []string{path}, nil)
	var protoErr model.ProtoError
	if !errors.As(err, &protoErr) {
		t.Fatalf("err = %v, want a ProtoError", err)
	}
	if protoErr.Path != path || protoErr.Line != 21 {
		t.Errorf("error at %s:%d, want %s:21 (%v)", protoErr.Path, protoErr.Line, path, err)
	}
	if !strings.Contains(protoErr.Message, "reflection/grpc_testing/proto2.proto") {
		t.Errorf("message %q does not name the missing import", protoErr.Message)
	}
}

func TestAParseErrorCarriesFileLineAndColumn(t *testing.T) {
	// health.proto cut after line 33, inside "message HealthCheckResponse {",
	// ends before the message closes.
	source, err := os.ReadFile(healthProto)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(source), "\n")
	path := filepath.Join(t.TempDir(), "health.proto")
	if err := os.WriteFile(path, []byte(strings.Join(lines[:33], "")), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = compileProtoFiles(grpcContext(t), "", []string{path}, nil)
	var protoErr model.ProtoError
	if !errors.As(err, &protoErr) {
		t.Fatalf("err = %v, want a ProtoError", err)
	}
	if protoErr.Path != path || protoErr.Line != 34 || protoErr.Col != 1 {
		t.Errorf("error at %s:%d:%d, want %s:34:1 (%v)", protoErr.Path, protoErr.Line, protoErr.Col, path, err)
	}
}

func TestAMissingProtoFileIsAnErrorNamingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.proto")
	_, err := compileProtoFiles(grpcContext(t), "", []string{path}, nil)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("err = %v, want one naming %s", err, path)
	}
}

func findCompiledService(t *testing.T, files linker.Files, name protoreflect.FullName) protoreflect.ServiceDescriptor {
	t.Helper()
	found, err := files.AsResolver().FindDescriptorByName(name)
	if err != nil {
		t.Fatal(err)
	}
	service, ok := found.(protoreflect.ServiceDescriptor)
	if !ok {
		t.Fatalf("%s is a %T, not a service", name, found)
	}
	return service
}
