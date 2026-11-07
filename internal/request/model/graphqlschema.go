package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// maxStarterFields bounds the fields a starter query selects, so a type
// with dozens of columns does not produce a page of query.
const maxStarterFields = 8

// Schema is what an introspection answer says a server can be asked: its
// query and mutation operations, each with a ready-to-edit query.
type Schema struct {
	Operations []Operation
}

// Operation is one root field of the schema.
type Operation struct {
	Label     string // "query user(id: ID!): User"
	Query     string // a query that calls it, selecting the simple fields of the result
	Variables string // a JSON object with a value for each argument; "" without arguments
}

type typeRef struct {
	Kind   string   `json:"kind"`
	Name   string   `json:"name"`
	OfType *typeRef `json:"ofType"`
}

type schemaField struct {
	Name string `json:"name"`
	Args []struct {
		Name string  `json:"name"`
		Type typeRef `json:"type"`
	} `json:"args"`
	Type typeRef `json:"type"`
}

type schemaType struct {
	Kind   string        `json:"kind"`
	Name   string        `json:"name"`
	Fields []schemaField `json:"fields"`
}

// ParseSchema reads the answer to IntrospectionQuery. Any other answer is
// an error, so a caller can try it on every GraphQL response.
func ParseSchema(body []byte) (*Schema, error) {
	var answer struct {
		Data struct {
			Schema *struct {
				QueryType    *struct{ Name string } `json:"queryType"`
				MutationType *struct{ Name string } `json:"mutationType"`
				Types        []schemaType           `json:"types"`
			} `json:"__schema"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &answer); err != nil {
		return nil, fmt.Errorf("parse introspection answer: %w", err)
	}
	raw := answer.Data.Schema
	if raw == nil {
		return nil, errors.New("the answer has no __schema")
	}
	types := map[string]schemaType{}
	for _, typ := range raw.Types {
		types[typ.Name] = typ
	}
	schema := &Schema{}
	roots := []struct {
		kind string
		name string
	}{{"query", ""}, {"mutation", ""}}
	if raw.QueryType != nil {
		roots[0].name = raw.QueryType.Name
	}
	if raw.MutationType != nil {
		roots[1].name = raw.MutationType.Name
	}
	for _, root := range roots {
		for _, field := range types[root.name].Fields {
			schema.Operations = append(schema.Operations, newOperation(root.kind, field, types))
		}
	}
	return schema, nil
}

func newOperation(kind string, field schemaField, types map[string]schemaType) Operation {
	var labelArgs, declared, passed, variables []string
	for _, arg := range field.Args {
		typeText := arg.Type.String()
		labelArgs = append(labelArgs, arg.Name+": "+typeText)
		declared = append(declared, "$"+arg.Name+": "+typeText)
		passed = append(passed, arg.Name+": $"+arg.Name)
		variables = append(variables, fmt.Sprintf("  %q: %s", arg.Name, arg.Type.starterValue()))
	}
	label := fmt.Sprintf("%s %s", kind, field.Name)
	call, header := field.Name, fmt.Sprintf("%s %s", kind, field.Name)
	if len(field.Args) > 0 {
		label += "(" + strings.Join(labelArgs, ", ") + ")"
		call += "(" + strings.Join(passed, ", ") + ")"
		header += "(" + strings.Join(declared, ", ") + ")"
	}
	op := Operation{Label: label + ": " + field.Type.String()}
	op.Query = header + " {\n  " + call + starterSelection(field.Type, types) + "\n}"
	if len(variables) > 0 {
		op.Variables = "{\n" + strings.Join(variables, ",\n") + "\n}"
	}
	return op
}

// String is the type as written in GraphQL: [User]!, ID.
func (ref typeRef) String() string {
	switch {
	case ref.Kind == "NON_NULL" && ref.OfType != nil:
		return ref.OfType.String() + "!"
	case ref.Kind == "LIST" && ref.OfType != nil:
		return "[" + ref.OfType.String() + "]"
	}
	return ref.Name
}

// named is the type with lists and non-null wrappers peeled off.
func (ref typeRef) named() typeRef {
	for ref.OfType != nil {
		ref = *ref.OfType
	}
	return ref
}

// starterValue is a JSON value to start a variable from: a zero of the
// scalar when the argument is required, null otherwise.
func (ref typeRef) starterValue() string {
	if ref.Kind != "NON_NULL" || ref.OfType == nil {
		return "null"
	}
	inner := *ref.OfType
	if inner.Kind != "SCALAR" {
		return "null"
	}
	switch inner.Name {
	case "Int", "Float":
		return "0"
	case "Boolean":
		return "false"
	}
	return `""`
}

// starterSelection is the selection set for a result of type ref: its
// scalar and enum fields, or __typename when it has none. Scalars need none.
func starterSelection(ref typeRef, types map[string]schemaType) string {
	target := types[ref.named().Name]
	if target.Kind != "OBJECT" && target.Kind != "INTERFACE" {
		return ""
	}
	var picked []string
	for _, field := range target.Fields {
		kind := field.Type.named().Kind
		if (kind == "SCALAR" || kind == "ENUM") && len(picked) < maxStarterFields {
			picked = append(picked, "    "+field.Name)
		}
	}
	if len(picked) == 0 {
		picked = []string{"    __typename"}
	}
	return " {\n" + strings.Join(picked, "\n") + "\n  }"
}

// IntrospectionQuery asks a GraphQL server for its schema: the types with
// their fields, arguments and kinds, enough to read what can be queried.
const IntrospectionQuery = `query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    types {
      kind name description
      fields(includeDeprecated: true) { name description args { name type { ...TypeRef } } type { ...TypeRef } }
      inputFields { name type { ...TypeRef } }
      enumValues(includeDeprecated: true) { name }
    }
  }
}
fragment TypeRef on __Type { kind name ofType { kind name ofType { kind name ofType { kind name } } } }`
