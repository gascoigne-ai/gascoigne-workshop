package proto

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// askReflection asks the server's reflection service one question. Servers
// that only have the older v1alpha service are asked that one: the two
// have the same messages on the wire, so the request is converted by
// encoding.
func askReflection(ctx context.Context, conn *grpc.ClientConn, request *reflectionv1.ServerReflectionRequest) (*reflectionv1.ServerReflectionResponse, error) {
	response, err := reflectV1(ctx, conn, request)
	if status.Code(err) != codes.Unimplemented {
		return response, err
	}
	return reflectV1Alpha(ctx, conn, request)
}

func reflectV1(ctx context.Context, conn *grpc.ClientConn, request *reflectionv1.ServerReflectionRequest) (*reflectionv1.ServerReflectionResponse, error) {
	stream, err := reflectionv1.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(request); err != nil {
		return nil, receiveError(stream.Recv, err)
	}
	response, err := stream.Recv()
	_ = stream.CloseSend()
	return response, err
}

func reflectV1Alpha(ctx context.Context, conn *grpc.ClientConn, request *reflectionv1.ServerReflectionRequest) (*reflectionv1.ServerReflectionResponse, error) {
	wire, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	var alphaRequest reflectionv1alpha.ServerReflectionRequest
	if err := proto.Unmarshal(wire, &alphaRequest); err != nil {
		return nil, err
	}
	stream, err := reflectionv1alpha.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&alphaRequest); err != nil {
		return nil, err
	}
	alphaResponse, err := stream.Recv()
	_ = stream.CloseSend()
	if err != nil {
		return nil, err
	}
	wire, err = proto.Marshal(alphaResponse)
	if err != nil {
		return nil, err
	}
	var response reflectionv1.ServerReflectionResponse
	if err := proto.Unmarshal(wire, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// receiveError is the reason a Send on a stream failed: the stream's own
// error, which Recv holds, rather than a bare io.EOF.
func receiveError(recv func() (*reflectionv1.ServerReflectionResponse, error), sendErr error) error {
	if sendErr != io.EOF {
		return sendErr
	}
	_, err := recv()
	return err
}

// listServices is the names of the services the server has.
func listServices(ctx context.Context, conn *grpc.ClientConn) ([]string, error) {
	response, err := askReflection(ctx, conn, &reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_ListServices{ListServices: ""},
	})
	if err != nil {
		return nil, fmt.Errorf("list services (does the server have reflection on?): %w", err)
	}
	list := response.GetListServicesResponse()
	if list == nil {
		return nil, fmt.Errorf("list services: %s", response.GetErrorResponse().GetErrorMessage())
	}
	names := make([]string, 0, len(list.Service))
	for _, service := range list.Service {
		names = append(names, service.Name)
	}
	return names, nil
}

// loadSymbol fetches the file that declares symbol and everything it
// imports, and returns them as a registry. Well-known types such as
// google/protobuf/any.proto are taken from this program's own copy when
// the server does not send them.
func loadSymbol(ctx context.Context, conn *grpc.ClientConn, symbol string) (*protoregistry.Files, error) {
	response, err := askReflection(ctx, conn, &reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: symbol},
	})
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", symbol, err)
	}
	if failure := response.GetErrorResponse(); failure != nil {
		return nil, fmt.Errorf("find %s: %s", symbol, failure.ErrorMessage)
	}
	files := map[string]*descriptorpb.FileDescriptorProto{}
	if err := addFiles(files, response.GetFileDescriptorResponse().GetFileDescriptorProto()); err != nil {
		return nil, err
	}
	for again := true; again; {
		again = false
		for _, file := range files {
			for _, dependency := range file.Dependency {
				if _, have := files[dependency]; have {
					continue
				}
				fetched, err := fetchFile(ctx, conn, dependency)
				if err != nil {
					return nil, err
				}
				files[dependency], again = fetched, true
			}
		}
	}
	set := &descriptorpb.FileDescriptorSet{}
	for _, file := range files {
		set.File = append(set.File, file)
	}
	registry, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, fmt.Errorf("read the descriptors of %s: %w", symbol, err)
	}
	return registry, nil
}

// addFiles decodes serialized file descriptors into files by name.
func addFiles(files map[string]*descriptorpb.FileDescriptorProto, serialized [][]byte) error {
	for _, raw := range serialized {
		file := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(raw, file); err != nil {
			return fmt.Errorf("decode a file descriptor: %w", err)
		}
		files[file.GetName()] = file
	}
	return nil
}

// fetchFile gets one imported file: this program's own copy when it is a
// well-known one, else the server's.
func fetchFile(ctx context.Context, conn *grpc.ClientConn, name string) (*descriptorpb.FileDescriptorProto, error) {
	if known, err := protoregistry.GlobalFiles.FindFileByPath(name); err == nil {
		return protodesc.ToFileDescriptorProto(known), nil
	}
	response, err := askReflection(ctx, conn, &reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileByFilename{FileByFilename: name},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", name, err)
	}
	if failure := response.GetErrorResponse(); failure != nil {
		return nil, fmt.Errorf("fetch %s: %s", name, failure.ErrorMessage)
	}
	files := map[string]*descriptorpb.FileDescriptorProto{}
	if err := addFiles(files, response.GetFileDescriptorResponse().GetFileDescriptorProto()); err != nil {
		return nil, err
	}
	file, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("the server did not send %s", name)
	}
	return file, nil
}

// findService looks a service up in the loaded files.
func findService(files *protoregistry.Files, name string) (protoreflect.ServiceDescriptor, error) {
	found, err := files.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, fmt.Errorf("the server has no service %s", name)
	}
	service, ok := found.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a service", name)
	}
	return service, nil
}
