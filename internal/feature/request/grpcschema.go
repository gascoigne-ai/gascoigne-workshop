package request

import (
	"encoding/json"
	"strings"
)

// MessageSchema describes a gRPC message: its fields in declaration order.
type MessageSchema struct {
	Name   string
	Fields []SchemaField
	// Cut says the message is already open further up the tree (a
	// recursive type), so its fields are left out here.
	Cut bool
}

// SchemaField is one field of a MessageSchema.
type SchemaField struct {
	Name       string
	Type       string // as the pane shows it: "string", "enum Status", "message Address", "map<string, int32>"
	Repeated   bool
	MapKey     string   // the key type of a map field, empty otherwise
	Oneof      string   // the oneof the field belongs to, empty otherwise
	EnumValues []string // in declaration order
	Message    *MessageSchema
}

// Skeleton is the JSON of a request message with every field at its
// default; only the first member of a oneof is written.
func (schema MessageSchema) Skeleton() string {
	var out strings.Builder
	schema.writeSkeleton(&out)
	return out.String()
}

// scalarDefaults are the skeleton values of the scalar types that are not
// numbers.
var scalarDefaults = map[string]string{"string": `""`, "bytes": `""`, "bool": "false"}

func (schema MessageSchema) writeSkeleton(out *strings.Builder) {
	out.WriteByte('{')
	seenOneofs := map[string]bool{}
	first := true
	for _, field := range schema.Fields {
		if field.Oneof != "" {
			if seenOneofs[field.Oneof] {
				continue
			}
			seenOneofs[field.Oneof] = true
		}
		if !first {
			out.WriteByte(',')
		}
		first = false
		name, _ := json.Marshal(field.Name)
		out.Write(name)
		out.WriteByte(':')
		field.writeSkeleton(out)
	}
	out.WriteByte('}')
}

func (field SchemaField) writeSkeleton(out *strings.Builder) {
	switch {
	case field.MapKey != "":
		out.WriteString("{}")
	case field.Repeated:
		out.WriteString("[]")
	case field.Message != nil:
		field.Message.writeSkeleton(out)
	case len(field.EnumValues) > 0:
		name, _ := json.Marshal(field.EnumValues[0])
		out.Write(name)
	default:
		def, ok := scalarDefaults[field.Type]
		if !ok {
			def = "0"
		}
		out.WriteString(def)
	}
}
