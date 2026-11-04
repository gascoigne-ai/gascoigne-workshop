package proto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// encodeGraphQL is the usual GraphQL-over-HTTP body: a JSON object with
// the query and, when there are any, the variables.
func encodeGraphQL(req model.Request) ([]byte, string, error) {
	if strings.TrimSpace(req.Body) == "" {
		return nil, "", errors.New("write a GraphQL query")
	}
	payload := map[string]any{"query": req.Body}
	if v := strings.TrimSpace(req.GraphQLVars); v != "" {
		var vars map[string]any
		if err := json.Unmarshal([]byte(v), &vars); err != nil {
			return nil, "", fmt.Errorf("the variables are not a JSON object: %w", err)
		}
		payload["variables"] = vars
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return nil, "", err
	}
	return bytes.TrimSpace(buf.Bytes()), "application/json", nil
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
