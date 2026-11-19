package proto

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// methodSource tells what services a gRPC server offers and describes
// them; calls are made from its descriptors whichever source it is.
type methodSource interface {
	// serviceNames lists the services there are to call.
	serviceNames(ctx context.Context) ([]string, error)
	// service describes one service by its full name.
	service(ctx context.Context, name string) (protoreflect.ServiceDescriptor, error)
}

// sourceKind is one row of the table of method sources.
type sourceKind struct {
	open func(ctx context.Context, conn *grpc.ClientConn, source model.ProtoSource) (methodSource, error)
	// asksServer says the source reads the server, so listing methods
	// needs a connection; files are read without one.
	asksServer bool
}

// methodSources opens a request's source, by model.ProtoSource.Kind.
var methodSources = map[model.SourceKind]sourceKind{
	model.SourceReflection: {open: openReflection, asksServer: true},
	model.SourceProtoFiles: {open: openProtoFiles},
}

// sourceKindOf is the kind of a request's source; ProtoSource.Kind names
// only kinds that methodSources holds.
func sourceKindOf(source model.ProtoSource) sourceKind {
	return methodSources[source.Kind()]
}

// openMethodSource opens the source of a request's methods. conn is the
// server's connection, which reflection asks and proto files ignore.
func openMethodSource(ctx context.Context, conn *grpc.ClientConn, source model.ProtoSource) (methodSource, error) {
	return sourceKindOf(source).open(ctx, conn, source)
}

// reflectionSource asks the server's reflection service.
type reflectionSource struct{ conn *grpc.ClientConn }

func openReflection(_ context.Context, conn *grpc.ClientConn, _ model.ProtoSource) (methodSource, error) {
	return reflectionSource{conn: conn}, nil
}

func (s reflectionSource) serviceNames(ctx context.Context) ([]string, error) {
	services, err := listServices(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	var callable []string
	for _, name := range services {
		if strings.HasPrefix(name, "grpc.reflection.") { // the means of listing, not a service to call
			continue
		}
		callable = append(callable, name)
	}
	return callable, nil
}

func (s reflectionSource) service(ctx context.Context, name string) (protoreflect.ServiceDescriptor, error) {
	files, err := loadSymbol(ctx, s.conn, name)
	if err != nil {
		return nil, err
	}
	return findService(files, name)
}

// protoFileSource reads the request's .proto files. Paths are absolute in
// a request (model.ProtoSource), so no base directory is needed here.
type protoFileSource struct{ files linker.Files }

func openProtoFiles(ctx context.Context, _ *grpc.ClientConn, source model.ProtoSource) (methodSource, error) {
	files, err := compileProtoFiles(ctx, "", source.Files, source.ImportPaths)
	if err != nil {
		return nil, err
	}
	return protoFileSource{files: files}, nil
}

// serviceNames lists the services the given files declare; services of
// their imports are left out, as protoc leaves them out of its output.
func (s protoFileSource) serviceNames(context.Context) ([]string, error) {
	var names []string
	for _, file := range s.files {
		for i := 0; i < file.Services().Len(); i++ {
			names = append(names, string(file.Services().Get(i).FullName()))
		}
	}
	return names, nil
}

func (s protoFileSource) service(_ context.Context, name string) (protoreflect.ServiceDescriptor, error) {
	found, err := s.files.AsResolver().FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, fmt.Errorf("the proto files have no service %s", name)
	}
	service, ok := found.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a service", name)
	}
	return service, nil
}
