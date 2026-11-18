package model

import "testing"

func FuzzParseSchema(f *testing.F) {
	f.Add([]byte(introspectionAnswer))
	f.Add([]byte(`{"data":{"__schema":{"types":[]}}}`))
	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = ParseSchema(body)
	})
}

func FuzzParseMCPLists(f *testing.F) {
	f.Add(`{"result":{"tools":[{"name":"a","inputSchema":{"type":"object","properties":{"x":{"type":"string"}}}}]}}`)
	f.Add(`{"result":{"resources":[{"uri":"file:///a","name":"a"}]}}`)
	f.Add(`{"result":{"prompts":[{"name":"p","arguments":[{"name":"x","required":true}]}]}}`)
	f.Fuzz(func(t *testing.T, message string) {
		for _, tool := range ParseTools(message) {
			_ = Skeleton(tool.Schema)
		}
		_ = ParseResources(message)
		for _, prompt := range ParsePrompts(message) {
			_ = prompt.ArgsJSON()
		}
	})
}

func FuzzParseServerInfo(f *testing.F) {
	f.Add(`{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"s","version":"1"}}`)
	f.Fuzz(func(t *testing.T, result string) {
		_, _ = ParseServerInfo(result)
	})
}

// A protocol, body type or auth type parsed from any text is one of the
// known values and parses again to itself.
func FuzzParseProtocol(f *testing.F) {
	f.Add("rest")
	f.Add("ws ")
	f.Add("")
	f.Fuzz(func(t *testing.T, text string) {
		protocol, err := ParseProtocol(text)
		if err != nil {
			return
		}
		if again, err := ParseProtocol(string(protocol)); err != nil || again != protocol {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, protocol, again, err)
		}
	})
}

func FuzzParseBodyType(f *testing.F) {
	f.Add("json")
	f.Add("")
	f.Add("xml")
	f.Fuzz(func(t *testing.T, text string) {
		bodyType, err := ParseBodyType(text)
		if err != nil {
			return
		}
		if again, err := ParseBodyType(string(bodyType)); err != nil || again != bodyType {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, bodyType, again, err)
		}
	})
}

func FuzzParseAuthType(f *testing.F) {
	f.Add("bearer")
	f.Add("none")
	f.Add("")
	f.Fuzz(func(t *testing.T, text string) {
		authType, err := ParseAuthType(text)
		if err != nil {
			return
		}
		if again, err := ParseAuthType(string(authType)); err != nil || again != authType {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, authType, again, err)
		}
	})
}
