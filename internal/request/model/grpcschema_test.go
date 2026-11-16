package model

import "testing"

func TestMessageSchemaSkeleton(t *testing.T) {
	address := &MessageSchema{Name: "Address", Fields: []SchemaField{
		{Name: "street", Type: "string"},
		{Name: "zip", Type: "int32"},
	}}
	tests := []struct {
		name   string
		schema MessageSchema
		want   string
	}{
		{"no fields", MessageSchema{Name: "Empty"}, `{}`},
		{"scalars take their zero value", MessageSchema{Fields: []SchemaField{
			{Name: "service", Type: "string"},
			{Name: "count", Type: "int64"},
			{Name: "ratio", Type: "double"},
			{Name: "on", Type: "bool"},
			{Name: "blob", Type: "bytes"},
		}}, `{"service":"","count":0,"ratio":0,"on":false,"blob":""}`},
		{"an enum takes its first value", MessageSchema{Fields: []SchemaField{
			{Name: "status", Type: "enum Status", EnumValues: []string{"UNKNOWN", "SERVING"}},
		}}, `{"status":"UNKNOWN"}`},
		{"a repeated field is an empty list", MessageSchema{Fields: []SchemaField{
			{Name: "tags", Type: "string", Repeated: true},
		}}, `{"tags":[]}`},
		{"a map is an empty object", MessageSchema{Fields: []SchemaField{
			{Name: "labels", Type: "map<string, string>", MapKey: "string"},
		}}, `{"labels":{}}`},
		{"a message field is expanded", MessageSchema{Fields: []SchemaField{
			{Name: "home", Type: "message Address", Message: address},
		}}, `{"home":{"street":"","zip":0}}`},
		{"only the first member of a oneof is filled", MessageSchema{Fields: []SchemaField{
			{Name: "id", Type: "string", Oneof: "who"},
			{Name: "name", Type: "string", Oneof: "who"},
			{Name: "age", Type: "int32"},
		}}, `{"id":"","age":0}`},
		{"a message cut off as recursive is an empty object", MessageSchema{Fields: []SchemaField{
			{Name: "next", Type: "message Node", Message: &MessageSchema{Name: "Node", Cut: true}},
		}}, `{"next":{}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.schema.Skeleton(); got != test.want {
				t.Errorf("Skeleton() = %s, want %s", got, test.want)
			}
		})
	}
}
