package proto

import (
	"context"
	"encoding/json"
	"fmt"
)

// answerServer replies to a request the server sent: it answers ping and
// roots/list (this client offers no roots), and refuses anything else with
// "method not found", so a server that waits for an answer is not left
// hanging.
func (s *mcpSession) answerServer(message rpcMessage) {
	reply := map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(*message.ID)}
	switch message.Method {
	case "ping":
		reply["result"] = map[string]any{}
	case "roots/list":
		reply["result"] = map[string]any{"roots": []any{}}
	case "sampling/createMessage":
		if s.sampler == nil {
			reply["error"] = map[string]any{"code": -32601, "message": "method not found: " + message.Method}
			break
		}
		s.sample(message, reply)
	default:
		reply["error"] = map[string]any{"code": -32601, "message": "method not found: " + message.Method}
	}
	b, err := json.Marshal(reply)
	if err == nil {
		err = s.write(context.Background(), "response", b)
	}
	if err != nil { // the server waits for this answer, so the user must see it never left
		s.note(fmt.Errorf("answer the server's %s: %w", message.Method, err))
	}
}
