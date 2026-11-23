package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol"
)

var samplingAsked = request.SamplingRequest{
	SystemPrompt: "Be brief",
	Messages:     []request.SamplingMessage{{Role: "user", Text: "Say hi"}},
}

func TestTheAppOffersSamplingToMCPServers(t *testing.T) {
	a := testApp(t)
	client, ok := a.reg["mcp"].(protocol.MCP)
	if !ok || client.Sampler == nil {
		t.Fatalf("the mcp protocol is %#v, it must have a Sampler", a.reg["mcp"])
	}
}
