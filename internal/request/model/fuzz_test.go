package model

import "testing"

func FuzzParseSchema(fuzz *testing.F) {
	fuzz.Add([]byte(introspectionAnswer))
	fuzz.Add([]byte(`{"data":{"__schema":{"types":[]}}}`))
	fuzz.Fuzz(func(t *testing.T, body []byte) {
		_, _ = ParseSchema(body)
	})
}

func FuzzParseMCPLists(fuzz *testing.F) {
	fuzz.Add(`{"result":{"tools":[{"name":"a","inputSchema":{"type":"object","properties":{"x":{"type":"string"}}}}]}}`)
	fuzz.Add(`{"result":{"resources":[{"uri":"file:///a","name":"a"}]}}`)
	fuzz.Add(`{"result":{"prompts":[{"name":"p","arguments":[{"name":"x","required":true}]}]}}`)
	fuzz.Fuzz(func(t *testing.T, message string) {
		for _, tool := range ParseTools(message) {
			_ = Skeleton(tool.Schema)
		}
		_ = ParseResources(message)
		for _, prompt := range ParsePrompts(message) {
			_ = prompt.ArgsJSON()
		}
	})
}

func FuzzParseServerInfo(fuzz *testing.F) {
	fuzz.Add(`{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"s","version":"1"}}`)
	fuzz.Fuzz(func(t *testing.T, result string) {
		_, _ = ParseServerInfo(result)
	})
}

// A protocol, body type or auth type parsed from any text is one of the
// known values and parses again to itself.
func FuzzParseProtocol(fuzz *testing.F) {
	fuzz.Add("rest")
	fuzz.Add("ws ")
	fuzz.Add("")
	fuzz.Fuzz(func(t *testing.T, text string) {
		protocol, err := ParseProtocol(text)
		if err != nil {
			return
		}
		if again, err := ParseProtocol(string(protocol)); err != nil || again != protocol {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, protocol, again, err)
		}
	})
}

func FuzzParseBodyType(fuzz *testing.F) {
	fuzz.Add("json")
	fuzz.Add("")
	fuzz.Add("xml")
	fuzz.Fuzz(func(t *testing.T, text string) {
		bodyType, err := ParseBodyType(text)
		if err != nil {
			return
		}
		if again, err := ParseBodyType(string(bodyType)); err != nil || again != bodyType {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, bodyType, again, err)
		}
	})
}

func FuzzParseAuthType(fuzz *testing.F) {
	fuzz.Add("bearer")
	fuzz.Add("none")
	fuzz.Add("")
	fuzz.Fuzz(func(t *testing.T, text string) {
		authType, err := ParseAuthType(text)
		if err != nil {
			return
		}
		if again, err := ParseAuthType(string(authType)); err != nil || again != authType {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, authType, again, err)
		}
	})
}
