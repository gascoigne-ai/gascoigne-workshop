package app

import (
	"encoding/json"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// listEvent is the event the MCP session emits for a JSON-RPC response to
// a list request: the text of the message and the typed payload.
func listEvent(text string) reqport.Event {
	var message struct{ Result json.RawMessage }
	_ = json.Unmarshal([]byte(text), &message) // the callers pass fixed, valid messages
	event := reqport.Event{Dir: reqport.Received, Kind: reqport.KindResponse, Text: text}
	switch {
	case strings.Contains(text, `"tools":`):
		tools, _ := model.DecodeTools(message.Result)
		event.Payload = reqport.ToolsList{Tools: tools}
	case strings.Contains(text, `"resources":`):
		resources, _ := model.DecodeResources(message.Result)
		event.Payload = reqport.ResourcesList{Resources: resources}
	case strings.Contains(text, `"prompts":`):
		prompts, _ := model.DecodePrompts(message.Result)
		event.Payload = reqport.PromptsList{Prompts: prompts}
	}
	return event
}
