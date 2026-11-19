package proto

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// answerServer replies to a request the server sent: it answers ping and
// roots/list (this client offers no roots), and refuses anything else with
// "method not found", so a server that waits for an answer is not left
// hanging.
func (s *mcpSession) answerServer(message rpcMessage) {
	id := *message.ID
	var reply rpcReply
	switch {
	case message.Method == "ping":
		reply = resultReply(id, struct{}{})
	case message.Method == "roots/list":
		reply = resultReply(id, rootsResult{Roots: []struct{}{}})
	case message.Method == "sampling/createMessage" && s.sampler != nil:
		reply = s.sample(message)
	default:
		reply = methodNotFound(id, message.Method)
	}
	b, err := json.Marshal(reply)
	if err == nil {
		err = s.write(context.Background(), port.KindResponse, b)
	}
	if err != nil { // the server waits for this answer, so the user must see it never left
		s.note(fmt.Errorf("answer the server's %s: %w", message.Method, err))
	}
}
