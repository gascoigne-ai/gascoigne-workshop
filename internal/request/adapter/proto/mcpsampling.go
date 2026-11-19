package proto

import (
	"context"
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// sampleRejected is the answer to a sampling request that was declined.
const sampleRejected = "User rejected sampling request"

// sample puts a sampling request to the Sampler and fills reply with its
// answer: a text message, or an error when the request is declined or
// cannot be read.
func (s *mcpSession) sample(message rpcMessage, reply map[string]any) {
	var params struct {
		Messages []struct {
			Role    string `json:"role"`
			Content struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
		SystemPrompt string `json:"systemPrompt"`
		MaxTokens    int    `json:"maxTokens"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		reply["error"] = map[string]any{"code": -32602, "message": "invalid sampling request: " + err.Error()}
		return
	}
	request := port.SamplingRequest{SystemPrompt: params.SystemPrompt, MaxTokens: params.MaxTokens}
	for _, message := range params.Messages {
		text := message.Content.Text
		if message.Content.Type != "text" {
			text = "[" + message.Content.Type + " content]"
		}
		request.Messages = append(request.Messages, port.SamplingMessage{Role: message.Role, Text: text})
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { // the session ending abandons the question
		select {
		case <-s.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	result, ok := s.sampler(ctx, request)
	if !ok {
		reply["error"] = map[string]any{"code": -1, "message": sampleRejected}
		return
	}
	reply["result"] = map[string]any{
		"role": "assistant", "content": map[string]any{"type": "text", "text": result.Text},
		"model": result.Model, "stopReason": "endTurn",
	}
}
