package proto

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// The registry and request.Protocols name the same protocols, and a protocol
// whose row keeps a connection can open one.
func TestTheRegistryHasOneImplementationPerProtocolRow(t *testing.T) {
	registry := NewRegistry()
	if len(registry) != len(request.Protocols) {
		t.Fatalf("%d implementations for %d protocols", len(registry), len(request.Protocols))
	}
	for _, row := range request.Protocols {
		implementation, ok := registry[row.Name]
		if !ok {
			t.Errorf("%s has no implementation", row.Name)
			continue
		}
		if _, streams := implementation.(request.Streamer); streams != (row.Connection != request.OneShot) {
			t.Errorf("%s streams: %v, but its row connects %v", row.Name, streams, row.Connection)
		}
	}
}

func TestRegistrySend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		io.WriteString(writer, "ok")
	}))
	defer srv.Close()

	tests := []struct {
		name    string
		req     request.Request
		wantErr bool
		wantMsg string // substring of Response.Err
		want    int    // Response.Status
	}{
		{"rest", request.Request{Protocol: "rest", URL: srv.URL}, false, "", 200},
		{"websocket", request.Request{Protocol: "websocket"}, false, "URL scheme", 0},
		{"mcp", request.Request{Protocol: "mcp"}, false, "not an http or https URL", 0},
		{"unknown", request.Request{Protocol: "carrier-pigeon"}, true, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewRegistry().Send(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if resp.Status != tt.want || !strings.Contains(resp.Err, tt.wantMsg) {
				t.Fatalf("resp = %+v, want status %d err containing %q", resp, tt.want, tt.wantMsg)
			}
		})
	}
}

func TestRESTSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		body, _ := io.ReadAll(call.Body)
		writer.Header().Set("X-Echo-Method", call.Method)
		writer.Header().Set("X-Echo-Auth", call.Header.Get("Authorization"))
		writer.Header().Set("X-Echo-Skipped", call.Header.Get("X-Off"))
		writer.WriteHeader(http.StatusCreated)
		writer.Write(body)
	}))
	defer srv.Close()

	resp, err := REST{}.Send(context.Background(), request.Request{
		Method: "post",
		URL:    srv.URL,
		Body:   `{"a":1}`,
		Headers: []request.KV{
			{Name: "Authorization", Value: "Bearer t", Enabled: true},
			{Name: "X-Off", Value: "no", Enabled: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 201 || resp.StatusText != "Created" {
		t.Errorf("status = %d %q", resp.Status, resp.StatusText)
	}
	if string(resp.Body) != `{"a":1}` || resp.Size != 7 {
		t.Errorf("body = %q size %d", resp.Body, resp.Size)
	}
	if resp.Duration <= 0 {
		t.Errorf("duration = %v", resp.Duration)
	}
	got := map[string]string{}
	for _, h := range resp.Headers {
		got[h.Name] = h.Value
	}
	want := map[string]string{"X-Echo-Method": "POST", "X-Echo-Auth": "Bearer t", "X-Echo-Skipped": ""}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("header %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestRESTSendFailures(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	t.Run("bad method is an error", func(t *testing.T) {
		if _, err := (REST{}).Send(context.Background(), request.Request{Method: "BAD METHOD", URL: "http://x"}); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("connection refused lands in Err", func(t *testing.T) {
		resp, err := REST{}.Send(context.Background(), request.Request{URL: closed.URL})
		if err != nil || resp.Err == "" || resp.Status != 0 {
			t.Fatalf("resp = %+v err = %v", resp, err)
		}
	})
	t.Run("cancelled context lands in Err", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		resp, err := REST{}.Send(ctx, request.Request{URL: closed.URL})
		if err != nil || resp.Err == "" {
			t.Fatalf("resp = %+v err = %v", resp, err)
		}
	})
}

// capture records what the server received.
type capture struct {
	method, contentType string
	body                []byte
}

func captureServer(t *testing.T) (*httptest.Server, *capture) {
	t.Helper()
	got := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		got.method, got.contentType = call.Method, call.Header.Get("Content-Type")
		got.body, _ = io.ReadAll(call.Body)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func TestRESTBodyTypes(t *testing.T) {
	form := []request.KV{
		{Name: "q", Value: "a b&c", Enabled: true},
		{Name: "off", Value: "x", Enabled: false},
		{Name: "", Value: "unnamed", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
	}
	tests := []struct {
		name            string
		req             request.Request
		wantBody        string
		wantContentType string
	}{
		{"none sends nothing", request.Request{BodyType: request.BodyNone, Body: "ignored"}, "", ""},
		{"legacy raw body has no content type", request.Request{Body: `{"a":1}`}, `{"a":1}`, ""},
		{"json", request.Request{BodyType: request.BodyJSON, Body: `{"a":1}`}, `{"a":1}`, "application/json"},
		{"text", request.Request{BodyType: request.BodyText, Body: "hello"}, "hello", "text/plain; charset=utf-8"},
		{"form skips disabled and unnamed rows and escapes", request.Request{BodyType: request.BodyForm, Form: form}, "q=a+b%26c&page=2", "application/x-www-form-urlencoded"},
		{"form ignores the text body", request.Request{BodyType: request.BodyForm, Body: "text", Form: form[:1]}, "q=a+b%26c", "application/x-www-form-urlencoded"},
		{"json body ignores form rows", request.Request{BodyType: request.BodyJSON, Body: "{}", Form: form}, "{}", "application/json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, got := captureServer(t)
			tt.req.Method, tt.req.URL = "POST", srv.URL
			if _, err := (REST{}).Send(context.Background(), tt.req); err != nil {
				t.Fatal(err)
			}
			if string(got.body) != tt.wantBody || got.contentType != tt.wantContentType {
				t.Fatalf("body %q type %q; want %q %q", got.body, got.contentType, tt.wantBody, tt.wantContentType)
			}
		})
	}
}

func TestRESTMultipartBody(t *testing.T) {
	srv, got := captureServer(t)
	_, err := REST{}.Send(context.Background(), request.Request{
		Method: "POST", URL: srv.URL, BodyType: request.BodyMultipart,
		Form: []request.KV{{Name: "a", Value: "1", Enabled: true}, {Name: "skip", Value: "x"}, {Name: "b", Value: "héllo", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mediaType, params, err := mime.ParseMediaType(got.contentType)
	if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
		t.Fatalf("content type %q (%v)", got.contentType, err)
	}
	reader := multipart.NewReader(bytes.NewReader(got.body), params["boundary"])
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		data, _ := io.ReadAll(part)
		fields[part.FormName()] = string(data)
	}
	if len(fields) != 2 || fields["a"] != "1" || fields["b"] != "héllo" {
		t.Fatalf("fields = %v", fields)
	}
}

func TestUserContentTypeHeaderWins(t *testing.T) {
	tests := []struct {
		name    string
		headers []request.KV
		want    string
	}{
		{"user header replaces the automatic one", []request.KV{{Name: "content-type", Value: "application/vnd.api+json", Enabled: true}}, "application/vnd.api+json"},
		{"a disabled header does not", []request.KV{{Name: "Content-Type", Value: "x/y", Enabled: false}}, "application/json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, got := captureServer(t)
			REST{}.Send(context.Background(), request.Request{Method: "POST", URL: srv.URL, BodyType: request.BodyJSON, Body: "{}", Headers: tt.headers})
			if got.contentType != tt.want {
				t.Fatalf("content type = %q, want %q", got.contentType, tt.want)
			}
		})
	}
}

func TestRESTRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) { io.WriteString(writer, "landed") }))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		http.Redirect(writer, call, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	tests := []struct {
		name        string
		noRedirects bool
		wantStatus  int
		wantBody    string
	}{
		{"followed by default", false, 200, "landed"},
		{"stopped at the first redirect", true, 302, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := REST{}.Send(context.Background(), request.Request{URL: redirector.URL, NoRedirects: tt.noRedirects})
			if err != nil || resp.Status != tt.wantStatus || !strings.Contains(string(resp.Body), tt.wantBody) {
				t.Fatalf("resp %+v err %v", resp, err)
			}
			if tt.noRedirects {
				var loc string
				for _, h := range resp.Headers {
					if h.Name == "Location" {
						loc = h.Value
					}
				}
				if loc != target.URL {
					t.Fatalf("the redirect target must be visible in the headers, Location = %q", loc)
				}
			}
		})
	}
}

func TestRESTExplainsCancelAndTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, _ := REST{}.Send(ctx, request.Request{URL: srv.URL})
	if resp.Err != request.ErrCancelled {
		t.Errorf("cancelled: Err = %q, want %q", resp.Err, request.ErrCancelled)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	resp, _ = REST{}.Send(ctx, request.Request{URL: srv.URL})
	if resp.Err != request.ErrTimedOut {
		t.Errorf("timeout: Err = %q, want %q", resp.Err, request.ErrTimedOut)
	}
}

func TestRESTAuthTypes(t *testing.T) {
	tests := []struct {
		name       string
		auth       request.Auth
		path       string
		wantHeader [2]string // name, value that must arrive ("" name: none)
		wantQuery  string    // raw query that must arrive
	}{
		{"none", request.Auth{}, "/p", [2]string{}, ""},
		{"explicit none", request.Auth{Type: request.AuthNone, Token: "ignored"}, "/p", [2]string{}, ""},
		{"bearer", request.Auth{Type: request.AuthBearer, Token: "abc"}, "/p", [2]string{"Authorization", "Bearer abc"}, ""},
		{"basic", request.Auth{Type: request.AuthBasic, User: "ann", Password: "pw:x"}, "/p", [2]string{"Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte("ann:pw:x"))}, ""},
		{"api key header", request.Auth{Type: request.AuthAPIKey, KeyName: "X-Api-Key", KeyValue: "k1"}, "/p", [2]string{"X-Api-Key", "k1"}, ""},
		{"api key header is the default place", request.Auth{Type: request.AuthAPIKey, KeyIn: request.KeyInHeader, KeyName: "X-Api-Key", KeyValue: "k1"}, "/p", [2]string{"X-Api-Key", "k1"}, ""},
		{"api key in the query", request.Auth{Type: request.AuthAPIKey, KeyIn: request.KeyInQuery, KeyName: "key", KeyValue: "a b"}, "/p", [2]string{}, "key=a+b"},
		{"api key keeps the existing query", request.Auth{Type: request.AuthAPIKey, KeyIn: request.KeyInQuery, KeyName: "key", KeyValue: "k"}, "/p?x=1", [2]string{}, "key=k&x=1"},
		{"api key without a name adds nothing", request.Auth{Type: request.AuthAPIKey, KeyValue: "k"}, "/p", [2]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotHeader http.Header
			var gotQuery string
			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
				gotHeader, gotQuery = call.Header.Clone(), call.URL.RawQuery
			}))
			defer srv.Close()
			_, err := REST{}.Send(context.Background(), request.Request{URL: srv.URL + tt.path, Auth: tt.auth})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantHeader[0] != "" && gotHeader.Get(tt.wantHeader[0]) != tt.wantHeader[1] {
				t.Errorf("%s = %q, want %q", tt.wantHeader[0], gotHeader.Get(tt.wantHeader[0]), tt.wantHeader[1])
			}
			if tt.wantHeader[0] == "" && (gotHeader.Get("Authorization") != "" || gotHeader.Get("X-Api-Key") != "") {
				t.Errorf("no credentials expected, got %v", gotHeader)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tt.wantQuery)
			}
		})
	}
}

func TestAuthOverridesAManualAuthorizationHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) { got = call.Header.Get("Authorization") }))
	defer srv.Close()
	REST{}.Send(context.Background(), request.Request{
		URL:     srv.URL,
		Headers: []request.KV{{Name: "Authorization", Value: "manual", Enabled: true}},
		Auth:    request.Auth{Type: request.AuthBearer, Token: "t"},
	})
	if got != "Bearer t" {
		t.Fatalf("Authorization = %q: the Auth pane wins over a header typed by hand", got)
	}
}

func TestUnknownAuthTypeIsAnError(t *testing.T) {
	if _, err := (REST{}).Send(context.Background(), request.Request{URL: "http://x", Auth: request.Auth{Type: "kerberos"}}); err == nil {
		t.Fatal("want an error for an unknown auth type")
	}
}

func TestConfiguredPassesTheNetworkSettingsToRESTAndTheSamplerToMCP(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	sampler := func(context.Context, request.SamplingRequest) (request.SamplingResult, bool) {
		return request.SamplingResult{}, false
	}
	registry := Configured(request.ProtocolConfig{Jar: jar, Proxy: "http://p", CACert: "ca", ClientCert: "cert", ClientKey: "key", Sampler: sampler})
	rest, ok := registry["rest"].(REST)
	if !ok || rest.Jar != jar || rest.Proxy != "http://p" || rest.CAFile != "ca" || rest.CertFile != "cert" || rest.KeyFile != "key" {
		t.Fatalf("rest = %+v", registry["rest"])
	}
	if mcp, ok := registry["mcp"].(MCP); !ok || mcp.Sampler == nil {
		t.Fatalf("mcp = %+v", registry["mcp"])
	}
	for _, name := range []request.Protocol{"websocket", "sse", "grpc"} {
		if registry[name] == nil {
			t.Errorf("%s is missing", name)
		}
	}
}
