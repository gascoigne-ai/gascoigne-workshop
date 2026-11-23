package window

import (
	"encoding/json"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// listEvent is the event the MCP session emits for a JSON-RPC response to
// a list request: the text of the message and the typed payload.
func listEvent(text string) request.Event {
	var message struct{ Result json.RawMessage }
	_ = json.Unmarshal([]byte(text), &message) // the callers pass fixed, valid messages
	event := request.Event{Dir: request.Received, Kind: request.KindResponse, Text: text}
	switch {
	case strings.Contains(text, `"tools":`):
		tools, _ := request.DecodeTools(message.Result)
		event.Payload = request.ToolsList{Tools: tools}
	case strings.Contains(text, `"resources":`):
		resources, _ := request.DecodeResources(message.Result)
		event.Payload = request.ResourcesList{Resources: resources}
	case strings.Contains(text, `"prompts":`):
		prompts, _ := request.DecodePrompts(message.Result)
		event.Payload = request.PromptsList{Prompts: prompts}
	}
	return event
}
