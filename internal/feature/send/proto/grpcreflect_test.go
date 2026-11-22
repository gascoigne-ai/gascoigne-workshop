package proto

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	protoV2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// endlessReflection answers every file request with a file that imports
// one more file, so a client that follows dependencies never finishes.
type endlessReflection struct {
	reflectionv1.UnimplementedServerReflectionServer
}

func (endlessReflection) ServerReflectionInfo(stream reflectionv1.ServerReflection_ServerReflectionInfoServer) error {
	request, err := stream.Recv()
	if err != nil {
		return err
	}
	name := request.GetFileByFilename()
	if name == "" {
		name = "f0.proto"
	}
	var number int
	fmt.Sscanf(name, "f%d.proto", &number)
	file := &descriptorpb.FileDescriptorProto{
		Name:       protoV2.String(name),
		Syntax:     protoV2.String("proto3"),
		Dependency: []string{fmt.Sprintf("f%d.proto", number+1)},
	}
	raw, _ := protoV2.Marshal(file)
	return stream.Send(&reflectionv1.ServerReflectionResponse{
		MessageResponse: &reflectionv1.ServerReflectionResponse_FileDescriptorResponse{
			FileDescriptorResponse: &reflectionv1.FileDescriptorResponse{FileDescriptorProto: [][]byte{raw}},
		},
	})
}

// loadSymbol followed dependencies for as long as the server named new
// ones, so a hostile or broken reflection server kept it fetching until
// the context ended.
func TestReflectionStopsFollowingAnEndlessDependencyChain(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	reflectionv1.RegisterServerReflectionServer(server, endlessReflection{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = loadSymbol(ctx, conn, "some.Symbol")
	if err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("err = %v, want a too-many-files error before the 5s deadline (ctx err %v)", err, ctx.Err())
	}
}
