package proto

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// GRPCInputSchema describes the request message of the method of req, from
// the server's reflection or the request's proto files.
func GRPCInputSchema(ctx context.Context, req request.Request) (request.MessageSchema, error) {
	kind := sourceKindOf(req.Proto)
	var conn *grpc.ClientConn
	var err error
	if kind.asksServer {
		if conn, err = dial(req); err != nil {
			return request.MessageSchema{}, err
		}
		defer conn.Close()
	}
	ctx, err = outgoing(ctx, req)
	if err != nil {
		return request.MessageSchema{}, err
	}
	method, err := resolveMethod(ctx, conn, req)
	if err != nil {
		return request.MessageSchema{}, err
	}
	return messageSchema(method.Input()), nil
}

// maxSchemaMessages bounds the messages one schema expands: descriptors
// come from a server, and a chain of types that each hold the next twice
// is exponential.
const maxSchemaMessages = 500

// kindNames are the names the schema shows for the scalar kinds.
var kindNames = map[protoreflect.Kind]string{
	protoreflect.BoolKind: "bool", protoreflect.StringKind: "string", protoreflect.BytesKind: "bytes",
	protoreflect.Int32Kind: "int32", protoreflect.Sint32Kind: "sint32", protoreflect.Sfixed32Kind: "sfixed32",
	protoreflect.Int64Kind: "int64", protoreflect.Sint64Kind: "sint64", protoreflect.Sfixed64Kind: "sfixed64",
	protoreflect.Uint32Kind: "uint32", protoreflect.Fixed32Kind: "fixed32",
	protoreflect.Uint64Kind: "uint64", protoreflect.Fixed64Kind: "fixed64",
	protoreflect.FloatKind: "float", protoreflect.DoubleKind: "double",
}

func messageSchema(message protoreflect.MessageDescriptor) request.MessageSchema {
	describer := &schemaDescriber{open: map[protoreflect.FullName]bool{}, budget: maxSchemaMessages}
	return describer.message(message)
}

// schemaDescriber walks a message's fields. open holds the messages above
// the one being described, so a type that contains itself is cut off at
// its second visit; budget is how many more messages may be expanded.
type schemaDescriber struct {
	open   map[protoreflect.FullName]bool
	budget int
}

func (d *schemaDescriber) message(message protoreflect.MessageDescriptor) request.MessageSchema {
	schema := request.MessageSchema{Name: string(message.FullName())}
	if d.open[message.FullName()] || d.budget <= 0 {
		schema.Cut = true
		return schema
	}
	d.budget--
	d.open[message.FullName()] = true
	defer delete(d.open, message.FullName())
	for i := 0; i < message.Fields().Len(); i++ {
		schema.Fields = append(schema.Fields, d.field(message.Fields().Get(i)))
	}
	return schema
}

func (d *schemaDescriber) field(field protoreflect.FieldDescriptor) request.SchemaField {
	described := request.SchemaField{Name: string(field.Name()), Repeated: field.IsList()}
	if oneof := field.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() {
		described.Oneof = string(oneof.Name())
	}
	if field.IsMap() {
		key, value := field.MapKey(), field.MapValue()
		described.MapKey = kindName(key)
		described.Type = "map<" + kindName(key) + ", " + kindName(value) + ">"
		described.Message, described.EnumValues = d.target(value)
		return described
	}
	described.Type = kindName(field)
	described.Message, described.EnumValues = d.target(field)
	return described
}

// fieldTarget is the message or the enum values a field of that kind
// refers to; a scalar has neither.
func (d *schemaDescriber) target(field protoreflect.FieldDescriptor) (*request.MessageSchema, []string) {
	if message := field.Message(); message != nil {
		schema := d.message(message)
		return &schema, nil
	}
	enum := field.Enum()
	if enum == nil {
		return nil, nil
	}
	values := make([]string, 0, enum.Values().Len())
	for i := 0; i < enum.Values().Len(); i++ {
		values = append(values, string(enum.Values().Get(i).Name()))
	}
	return nil, values
}

func kindName(field protoreflect.FieldDescriptor) string {
	switch {
	case field.Message() != nil:
		return "message " + string(field.Message().Name())
	case field.Enum() != nil:
		return "enum " + string(field.Enum().Name())
	}
	return kindNames[field.Kind()]
}
