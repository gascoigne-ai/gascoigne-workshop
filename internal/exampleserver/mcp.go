package exampleserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const (
	fallbackMCPVersion = "2025-06-18"
	codeMethodMissing  = -32601
	codeBadParams      = -32602
)

type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// tool is one row of the table of MCP tools.
type tool struct {
	name        string
	description string
	schema      map[string]any
	call        func(arguments json.RawMessage) (string, error)
}

var tools = []tool{
	{
		name: "echo", description: "Returns its text argument",
		schema: objectSchema(map[string]any{"text": map[string]any{"type": "string"}}, "text"),
		call: func(arguments json.RawMessage) (string, error) {
			var input struct{ Text string }
			err := json.Unmarshal(arguments, &input)
			return input.Text, err
		},
	},
	{
		name: "add", description: "Adds two numbers",
		schema: objectSchema(map[string]any{"a": map[string]any{"type": "number"}, "b": map[string]any{"type": "number"}}, "a", "b"),
		call: func(arguments json.RawMessage) (string, error) {
			var input struct{ A, B float64 }
			if err := json.Unmarshal(arguments, &input); err != nil {
				return "", err
			}
			return strconv.FormatFloat(input.A+input.B, 'f', -1, 64), nil
		},
	},
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required}
}

// mcpHandler is a stateless streamable-HTTP MCP server: each POST is one
// JSON-RPC message answered as plain JSON, a notification gets 202.
func mcpHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodDelete:
			return // ends a session; there is none to end
		case http.MethodPost:
		default:
			writer.Header().Set("Allow", "POST, DELETE")
			http.Error(writer, "use POST", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		var message rpcRequest
		if err := json.Unmarshal(body, &message); err != nil {
			http.Error(writer, "not a JSON-RPC message: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(message.ID) == 0 {
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(answer(message)) // the client went away; nobody to tell
	})
}

func answer(message rpcRequest) rpcResponse {
	response := rpcResponse{JSONRPC: "2.0", ID: message.ID}
	switch message.Method {
	case "initialize":
		response.Result = initializeResult(message.Params)
	case "ping":
		response.Result = map[string]any{}
	case "tools/list":
		response.Result = map[string]any{"tools": toolList()}
	case "tools/call":
		response.Result, response.Error = callTool(message.Params)
	default:
		response.Error = &rpcError{codeMethodMissing, "method not found: " + message.Method}
	}
	return response
}

// initializeResult agrees to whatever protocol version the client asks
// for, which is all a stateless example server needs.
func initializeResult(params json.RawMessage) map[string]any {
	var asked struct{ ProtocolVersion string }
	_ = json.Unmarshal(params, &asked) // no params: use the fallback
	if asked.ProtocolVersion == "" {
		asked.ProtocolVersion = fallbackMCPVersion
	}
	return map[string]any{
		"protocolVersion": asked.ProtocolVersion,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      map[string]any{"name": "gascoigne-example", "version": "1"},
	}
}

func toolList() []map[string]any {
	list := make([]map[string]any, len(tools))
	for i, tool := range tools {
		list[i] = map[string]any{"name": tool.name, "description": tool.description, "inputSchema": tool.schema}
	}
	return list
}

func callTool(params json.RawMessage) (any, *rpcError) {
	var call struct {
		Name      string
		Arguments json.RawMessage
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &rpcError{codeBadParams, "bad params: " + err.Error()}
	}
	for _, tool := range tools {
		if tool.name != call.Name {
			continue
		}
		text, err := tool.call(call.Arguments)
		if err != nil {
			return toolResult(fmt.Sprintf("bad arguments: %v", err), true), nil
		}
		return toolResult(text, false), nil
	}
	return nil, &rpcError{codeBadParams, "unknown tool: " + call.Name}
}

func toolResult(text string, failed bool) map[string]any {
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": text}},
		"isError": failed,
	}
}
