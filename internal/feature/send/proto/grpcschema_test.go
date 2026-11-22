package proto

import (
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestGRPCInputSchemaOfAProtoFileMethod(t *testing.T) {
	call := fromHealthProto(t, grpcRequest("", "grpc.health.v1.Health/Check", ""))
	schema, err := GRPCInputSchema(grpcContext(t), call)
	if err != nil {
		t.Fatal(err)
	}
	want := request.MessageSchema{Name: healthCheckName, Fields: []request.SchemaField{{Name: "service", Type: "string"}}}
	if !reflect.DeepEqual(schema, want) {
		t.Errorf("schema = %+v, want %+v", schema, want)
	}
}

func TestGRPCInputSchemaNeedsAMethod(t *testing.T) {
	call := fromHealthProto(t, grpcRequest("", "", ""))
	if _, err := GRPCInputSchema(grpcContext(t), call); err == nil {
		t.Error("a request with no method has a schema")
	}
}

// binarylog.proto has nesting, a oneof, an enum, repeated fields and a
// message that others refer to.
func TestMessageSchemaOfBinarylogEntry(t *testing.T) {
	files, err := compileProtoFiles(grpcContext(t), "", []string{absolute(t, binarylogProto)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	found, err := files.AsResolver().FindDescriptorByName("grpc.binarylog.v1.GrpcLogEntry")
	if err != nil {
		t.Fatal(err)
	}
	schema := messageSchema(found.(protoreflect.MessageDescriptor))
	fields := fieldsByName(schema)
	if got := fields["type"]; got.Type != "enum EventType" || len(got.EnumValues) == 0 || got.EnumValues[0] != "EVENT_TYPE_UNKNOWN" {
		t.Errorf("type = %+v, want an enum starting at EVENT_TYPE_UNKNOWN", got)
	}
	if got := fields["client_header"]; got.Oneof != "payload" || got.Type != "message ClientHeader" || got.Message == nil {
		t.Errorf("client_header = %+v, want a message in oneof payload", got)
	}
	if got := fields["payload_truncated"]; got.Type != "bool" || got.Oneof != "" {
		t.Errorf("payload_truncated = %+v, want a plain bool", got)
	}
	metadata := fields["client_header"].Message
	entries := fieldsByName(*fieldsByName(*metadata)["metadata"].Message)["entry"]
	if !entries.Repeated || entries.Message == nil {
		t.Errorf("client_header.metadata.entry = %+v, want a repeated message", entries)
	}
}

// Struct holds Values that hold Structs: the second visit is cut off, or
// the schema would never end.
func TestMessageSchemaCutsARecursiveMessage(t *testing.T) {
	schema := messageSchema((&structpb.Struct{}).ProtoReflect().Descriptor())
	fields := fieldsByName(schema)["fields"]
	if fields.MapKey != "string" || fields.Message == nil {
		t.Fatalf("fields = %+v, want a map of Value", fields)
	}
	value := fieldsByName(*fields.Message)["struct_value"]
	if value.Message == nil || !value.Message.Cut {
		t.Errorf("struct_value = %+v, want a message cut off as recursive", value)
	}
}

func fieldsByName(schema request.MessageSchema) map[string]request.SchemaField {
	fields := map[string]request.SchemaField{}
	for _, field := range schema.Fields {
		fields[field.Name] = field
	}
	return fields
}

// A server's descriptors are untrusted: 40 messages that each hold the
// next one twice are 2^40 fields when expanded, so the schema stops
// growing at maxSchemaMessages.
func TestMessageSchemaOfAWideChainStaysBounded(t *testing.T) {
	const length = 40
	file := &descriptorpb.FileDescriptorProto{Name: proto.String("chain.proto"), Syntax: proto.String("proto3"), Package: proto.String("chain")}
	for i := 0; i < length; i++ {
		message := &descriptorpb.DescriptorProto{Name: proto.String(fmt.Sprintf("M%d", i))}
		if i+1 < length {
			for _, name := range []string{"left", "right"} {
				message.Field = append(message.Field, &descriptorpb.FieldDescriptorProto{
					Name: proto.String(name), JsonName: proto.String(name), Number: proto.Int32(int32(len(message.Field) + 1)),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(fmt.Sprintf(".chain.M%d", i+1)),
				})
			}
		}
		file.MessageType = append(file.MessageType, message)
	}
	built, err := protodesc.NewFile(file, nil)
	if err != nil {
		t.Fatal(err)
	}
	schema := messageSchema(built.Messages().Get(0))
	if got := countMessages(schema); got > maxSchemaMessages {
		t.Errorf("schema holds %d messages, want at most %d", got, maxSchemaMessages)
	}
}

// countMessages counts the messages a schema expands, not the cut ones.
func countMessages(schema request.MessageSchema) int {
	count := 1
	if schema.Cut {
		count = 0
	}
	for _, field := range schema.Fields {
		if field.Message != nil {
			count += countMessages(*field.Message)
		}
	}
	return count
}
