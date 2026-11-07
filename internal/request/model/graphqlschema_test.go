package model

import "testing"

// introspectionAnswer is shaped like the answer to IntrospectionQuery,
// trimmed to a few types (written from the GraphQL specification, not
// captured from a server).
const introspectionAnswer = `{"data":{"__schema":{
 "queryType":{"name":"Query"},"mutationType":{"name":"Mutation"},
 "types":[
  {"kind":"OBJECT","name":"Query","fields":[
    {"name":"user","args":[{"name":"id","type":{"kind":"NON_NULL","name":null,"ofType":{"kind":"SCALAR","name":"ID","ofType":null}}}],
     "type":{"kind":"OBJECT","name":"User","ofType":null}},
    {"name":"users","args":[],"type":{"kind":"NON_NULL","name":null,"ofType":{"kind":"LIST","name":null,"ofType":{"kind":"OBJECT","name":"User","ofType":null}}}},
    {"name":"version","args":[],"type":{"kind":"SCALAR","name":"String","ofType":null}}]},
  {"kind":"OBJECT","name":"Mutation","fields":[
    {"name":"rename","args":[
      {"name":"id","type":{"kind":"NON_NULL","name":null,"ofType":{"kind":"SCALAR","name":"ID","ofType":null}}},
      {"name":"name","type":{"kind":"SCALAR","name":"String","ofType":null}}],
     "type":{"kind":"OBJECT","name":"User","ofType":null}}]},
  {"kind":"OBJECT","name":"User","fields":[
    {"name":"id","args":[],"type":{"kind":"NON_NULL","name":null,"ofType":{"kind":"SCALAR","name":"ID","ofType":null}}},
    {"name":"name","args":[],"type":{"kind":"SCALAR","name":"String","ofType":null}},
    {"name":"role","args":[],"type":{"kind":"ENUM","name":"Role","ofType":null}},
    {"name":"friends","args":[],"type":{"kind":"LIST","name":null,"ofType":{"kind":"OBJECT","name":"User","ofType":null}}}]},
  {"kind":"ENUM","name":"Role","enumValues":[{"name":"ADMIN"}]}
 ]}}}`

func TestParseSchemaListsQueryAndMutationOperations(t *testing.T) {
	schema, err := ParseSchema([]byte(introspectionAnswer))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, op := range schema.Operations {
		got = append(got, op.Label)
	}
	want := []string{"query user(id: ID!): User", "query users: [User]!", "query version: String", "mutation rename(id: ID!, name: String): User"}
	if len(got) != len(want) {
		t.Fatalf("labels = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("label %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOperationsComeWithAStarterQueryAndVariables(t *testing.T) {
	schema, _ := ParseSchema([]byte(introspectionAnswer))
	tests := []struct {
		label, query, variables string
	}{
		{"query user(id: ID!): User", "query user($id: ID!) {\n  user(id: $id) {\n    id\n    name\n    role\n  }\n}", "{\n  \"id\": \"\"\n}"},
		{"query version: String", "query version {\n  version\n}", ""},
		{"mutation rename(id: ID!, name: String): User", "mutation rename($id: ID!, $name: String) {\n  rename(id: $id, name: $name) {\n    id\n    name\n    role\n  }\n}", "{\n  \"id\": \"\",\n  \"name\": null\n}"},
	}
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			for _, op := range schema.Operations {
				if op.Label != test.label {
					continue
				}
				if op.Query != test.query {
					t.Errorf("query =\n%s\nwant\n%s", op.Query, test.query)
				}
				if op.Variables != test.variables {
					t.Errorf("variables = %q, want %q", op.Variables, test.variables)
				}
				return
			}
			t.Fatal("operation not found")
		})
	}
}

func TestParseSchemaRefusesAnswersThatAreNotIntrospection(t *testing.T) {
	for _, body := range []string{`{"data":{"user":{"id":"1"}}}`, `not json`, `{"errors":[{"message":"no"}]}`} {
		if _, err := ParseSchema([]byte(body)); err == nil {
			t.Errorf("%q parsed as a schema", body)
		}
	}
}
