package proto

import (
	"context"
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// sampleRejected is the answer to a sampling request that was declined.
const sampleRejected = "User rejected sampling request"

// sampleBusy is the answer to a sampling request that comes while the user
// has another one open.
const sampleBusy = "another sampling request is open"

// sample puts a sampling request to the Sampler and answers with its
// text message, or with an error when the request is declined or cannot
// be read.
func (s *mcpSession) sample(message rpcMessage) rpcReply {
	if !s.sampling.tryTake(maxSamplePrompts) {
		return errorReply(*message.ID, rpcRejected, sampleBusy)
	}
	defer s.sampling.release()
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
	call := request.SamplingRequest{SystemPrompt: params.SystemPrompt, MaxTokens: params.MaxTokens}
	for _, message := range params.Messages {
		text := message.Content.Text
		if message.Content.Type != "text" {
			text = "[" + message.Content.Type + " content]"
		}
		call.Messages = append(call.Messages, request.SamplingMessage{Role: message.Role, Text: text})
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
	result, ok := s.sampler(ctx, call)
	if !ok {
		return errorReply(*message.ID, rpcRejected, sampleRejected)
	}
	return resultReply(*message.ID, samplingResult{
		Content: samplingContent{Type: "text", Text: result.Text},
		Model:   result.Model, Role: "assistant", StopReason: "endTurn",
	})
}
