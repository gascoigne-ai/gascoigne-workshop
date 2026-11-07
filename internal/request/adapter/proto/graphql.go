package proto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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
