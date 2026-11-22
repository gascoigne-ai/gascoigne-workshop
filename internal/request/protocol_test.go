package request

import (
	"reflect"
	"testing"
)

func TestParseProtocolAcceptsEachProtocolName(t *testing.T) {
	for _, want := range []Protocol{ProtocolREST, ProtocolWebSocket, ProtocolMCP, ProtocolSSE, ProtocolGRPC} {
		got, err := ParseProtocol(string(want))
		if err != nil || got != want {
			t.Errorf("ParseProtocol(%q) = %q, %v; want %q", want, got, err, want)
		}
	}
}

func TestParseProtocolRejectsAnUnknownOrEmptyName(t *testing.T) {
	for _, text := range []string{"REST", "http", "ws ", "\x00", ""} {
		if got, err := ParseProtocol(text); err == nil {
			t.Errorf("ParseProtocol(%q) = %q, want an error", text, got)
		}
	}
}

func TestParseBodyTypeAcceptsEachBodyType(t *testing.T) {
	for _, want := range []BodyType{BodyRaw, BodyNone, BodyJSON, BodyText, BodyForm, BodyMultipart, BodyGraphQL, BodyFile} {
		got, err := ParseBodyType(string(want))
		if err != nil || got != want {
			t.Errorf("ParseBodyType(%q) = %q, %v; want %q", want, got, err, want)
		}
	}
}

func TestParseBodyTypeRejectsAnUnknownName(t *testing.T) {
	for _, text := range []string{"JSON", "xml", "json ", "\x00"} {
		if got, err := ParseBodyType(text); err == nil {
			t.Errorf("ParseBodyType(%q) = %q, want an error", text, got)
		}
	}
}

func TestParseAuthTypeAcceptsEachAuthType(t *testing.T) {
	tests := []struct {
		text string
		want AuthType
	}{
		{"", AuthNone},
		{"none", AuthNone},
		{"bearer", AuthBearer},
		{"basic", AuthBasic},
		{"apikey", AuthAPIKey},
	}
	for _, test := range tests {
		got, err := ParseAuthType(test.text)
		if err != nil || got != test.want {
			t.Errorf("ParseAuthType(%q) = %q, %v; want %q", test.text, got, err, test.want)
		}
	}
}

func TestParseAuthTypeRejectsAnUnknownName(t *testing.T) {
	for _, text := range []string{"Bearer", "oauth2", "basic ", "\x00"} {
		if got, err := ParseAuthType(text); err == nil {
			t.Errorf("ParseAuthType(%q) = %q, want an error", text, got)
		}
	}
}

func TestParseKeyIn(t *testing.T) {
	cases := []struct {
		text    string
		want    KeyIn
		wantErr bool
	}{
		{"", "", false}, // the zero value stays, so a saved file keeps its bytes
		{"header", KeyInHeader, false},
		{"query", KeyInQuery, false},
		{"cookie", "", true},
		{"Header", "", true},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			got, err := ParseKeyIn(c.text)
			if got != c.want || (err != nil) != c.wantErr {
				t.Fatalf("ParseKeyIn(%q) = %q, %v; want %q, error %v", c.text, got, err, c.want, c.wantErr)
			}
		})
	}
}

func TestProtocolsListEachProtocolOnceInChipOrder(t *testing.T) {
	want := []struct {
		name  Protocol
		label string
	}{
		{ProtocolREST, "REST"}, {ProtocolWebSocket, "WS"}, {ProtocolMCP, "MCP"}, {ProtocolSSE, "SSE"}, {ProtocolGRPC, "gRPC"},
	}
	if len(Protocols) != len(want) {
		t.Fatalf("%d protocols, want %d", len(Protocols), len(want))
	}
	for i, row := range Protocols {
		if row.Name != want[i].name || row.Label != want[i].label {
			t.Errorf("row %d = %s %q, want %s %q", i, row.Name, row.Label, want[i].name, want[i].label)
		}
		if got := row.Name.Info(); !reflect.DeepEqual(got, row) {
			t.Errorf("%s.Info() = %+v, want its row %+v", row.Name, got, row)
		}
	}
}

func TestEachProtocolSaysHowItConnectsAndWhatItsMethodIs(t *testing.T) {
	tests := []struct {
		protocol     Protocol
		connection   Connection
		method       MethodKind
		runsCommand  bool
		resumeHeader string
	}{
		{ProtocolREST, OneShot, MethodHTTP, false, ""},
		{ProtocolWebSocket, Always, MethodNone, false, ""},
		{ProtocolMCP, Always, MethodNone, true, ""},
		{ProtocolSSE, Always, MethodNone, false, "Last-Event-ID"},
		{ProtocolGRPC, OnStream, MethodRPC, false, ""},
	}
	for _, test := range tests {
		info := test.protocol.Info()
		got := []any{info.Connection, info.Method, info.RunsCommand, info.ResumeHeader}
		want := []any{test.connection, test.method, test.runsCommand, test.resumeHeader}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: connection, method, command, resume = %v, want %v", test.protocol, got, want)
		}
	}
}

func TestNoProtocolHasTheZeroInfo(t *testing.T) {
	if got := Protocol("").Info(); got != (ProtocolInfo{}) {
		t.Fatalf("Info of no protocol = %+v, want the zero value", got)
	}
}
