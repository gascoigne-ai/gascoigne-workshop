package proto

import (
	"context"
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// sampleRejected is the answer to a sampling request that was declined.
const sampleRejected = "User rejected sampling request"

// sample puts a sampling request to the Sampler and answers with its
// text message, or with an error when the request is declined or cannot
// be read.
func (s *mcpSession) sample(message rpcMessage) rpcReply {
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
		return errorReply(*message.ID, rpcInvalidParams, "invalid sampling request: "+err.Error())
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
		return errorReply(*message.ID, rpcRejected, sampleRejected)
	}
	return resultReply(*message.ID, samplingResult{
		Content: samplingContent{Type: "text", Text: result.Text},
		Model:   result.Model, Role: "assistant", StopReason: "endTurn",
	})
}
