package proto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
)

// seenRequest is what a test server noted about one request.
type seenRequest struct {
	Header     http.Header
	Host       string
	URL        string
	RawQuery   string
	ClientCert string // common name of the client certificate, if any
}

type seenLog struct {
	mu       sync.Mutex
	requests []seenRequest
}

func (log *seenLog) note(call *http.Request) {
	entry := seenRequest{Header: call.Header.Clone(), Host: call.Host, URL: call.URL.String(), RawQuery: call.URL.RawQuery}
	if call.TLS != nil && len(call.TLS.PeerCertificates) > 0 {
		entry.ClientCert = call.TLS.PeerCertificates[0].Subject.CommonName
	}
	log.mu.Lock()
	log.requests = append(log.requests, entry)
	log.mu.Unlock()
}

func (log *seenLog) all() []seenRequest {
	log.mu.Lock()
	defer log.mu.Unlock()
	return append([]seenRequest(nil), log.requests...)
}

// first waits for the first request: a streaming connect returns after
// the server answered, but a failing one may never reach the server.
func (log *seenLog) first(t *testing.T) seenRequest {
	t.Helper()
	all := log.all()
	if len(all) == 0 {
		t.Fatal("the server saw no request")
	}
	return all[0]
}

// httpProtocol is one HTTP-based protocol with a server that speaks it.
type httpProtocol struct {
	name     string
	protocol request.Protocol
	handler  func(log *seenLog) http.Handler
	wsScheme bool // the URL scheme is ws or wss
}

var (
	restProtocol = httpProtocol{name: "rest", protocol: request.ProtocolREST, handler: func(log *seenLog) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
			log.note(call)
			writer.Write([]byte("ok"))
		})
	}}
	sseProtocol = httpProtocol{name: "sse", protocol: request.ProtocolSSE, handler: func(log *seenLog) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
			log.note(call)
			writer.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(writer, "data: hi\n\n")
		})
	}}
	mcpProtocol = httpProtocol{name: "mcp", protocol: request.ProtocolMCP, handler: func(log *seenLog) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
			log.note(call)
			if call.Method != http.MethodPost {
				writer.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			body, _ := io.ReadAll(call.Body)
			var message map[string]any
			json.Unmarshal(body, &message)
			out := mcpReply(message)
			if out == nil {
				writer.WriteHeader(http.StatusAccepted)
				return
			}
			encoded, _ := json.Marshal(out)
			writer.Header().Set("Content-Type", "application/json")
			writer.Write(encoded)
		})
	}}
	wsProtocol = httpProtocol{name: "websocket", protocol: request.ProtocolWebSocket, wsScheme: true, handler: func(log *seenLog) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
			log.note(call)
			conn, err := ws.Upgrade(writer, call)
			if err != nil {
				return
			}
			conn.Close(ws.CloseNormal, "")
		})
	}}

	streamers = []httpProtocol{sseProtocol, mcpProtocol, wsProtocol}
	allFour   = []httpProtocol{restProtocol, sseProtocol, mcpProtocol, wsProtocol}
)

// serve starts the protocol's server, over TLS when asked, and returns the
// URL a request for it uses. requireClientCert makes the server ask for one.
func (p httpProtocol) serve(t *testing.T, secure, requireClientCert bool) (*httptest.Server, *seenLog) {
	t.Helper()
	log := &seenLog{}
	srv := httptest.NewUnstartedServer(p.handler(log))
	if requireClientCert {
		srv.TLS = &tls.Config{ClientAuth: tls.RequireAnyClientCert}
	}
	if secure {
		srv.StartTLS()
	} else {
		srv.Start()
	}
	t.Cleanup(srv.Close)
	return srv, log
}

func (p httpProtocol) url(srv *httptest.Server) string {
	if p.wsScheme {
		return "ws" + strings.TrimPrefix(srv.URL, "http")
	}
	return srv.URL
}

// attempt connects (or sends) with a registry made from config and returns
// the failure, if any.
func (p httpProtocol) attempt(config request.ProtocolConfig, call request.Request) error {
	call.Protocol = p.protocol
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	registry := Configured(config)
	if p.protocol == request.ProtocolREST {
		response, err := registry.Send(ctx, call)
		if err != nil {
			return err
		}
		if response.Err != "" {
			return &textError{response.Err}
		}
		return nil
	}
	session, err := registry.Connect(ctx, call)
	if err != nil {
		return err
	}
	session.Close(context.Background())
	return nil
}

type textError struct{ text string }

func (e *textError) Error() string { return e.text }

func caFileOf(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	return writePEM(t, "ca.pem", srv.Certificate().Raw)
}

// writeClientKeyPair makes a self-signed client certificate and key in
// temp files.
func writeClientKeyPair(t *testing.T, commonName string) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: commonName}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	certFile = writePEM(t, "client.pem", der)
	keyFile = filepath.Join(t.TempDir(), "client.key")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func TestHTTPProtocolsRejectUntrustedCertificate(t *testing.T) {
	for _, p := range streamers {
		t.Run(p.name, func(t *testing.T) {
			srv, log := p.serve(t, true, false)
			err := p.attempt(request.ProtocolConfig{}, request.Request{URL: p.url(srv)})
			if err == nil || !strings.Contains(err.Error(), "certificate") {
				t.Fatalf("an unknown certificate must be refused, got %v", err)
			}
			if got := len(log.all()); got != 0 {
				t.Fatalf("the server saw %d requests", got)
			}
		})
	}
}

func TestHTTPProtocolsTrustConfiguredCAFile(t *testing.T) {
	for _, p := range streamers {
		t.Run(p.name, func(t *testing.T) {
			srv, _ := p.serve(t, true, false)
			err := p.attempt(request.ProtocolConfig{CACert: caFileOf(t, srv)}, request.Request{URL: p.url(srv)})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHTTPProtocolsHonorInsecureTLS(t *testing.T) {
	for _, p := range streamers {
		t.Run(p.name, func(t *testing.T) {
			srv, _ := p.serve(t, true, false)
			err := p.attempt(request.ProtocolConfig{}, request.Request{URL: p.url(srv), InsecureTLS: true})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHTTPProtocolsSendClientCertificate(t *testing.T) {
	certFile, keyFile := writeClientKeyPair(t, "workshop-client")
	for _, p := range streamers {
		t.Run(p.name, func(t *testing.T) {
			srv, log := p.serve(t, true, true)
			config := request.ProtocolConfig{ClientCert: certFile, ClientKey: keyFile}
			if err := p.attempt(config, request.Request{URL: p.url(srv), InsecureTLS: true}); err != nil {
				t.Fatal(err)
			}
			if got := log.first(t).ClientCert; got != "workshop-client" {
				t.Fatalf("the server saw client certificate %q", got)
			}
		})
	}
}

func TestHTTPProtocolsRejectUnreadableCAFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pem")
	for _, p := range streamers {
		t.Run(p.name, func(t *testing.T) {
			srv, _ := p.serve(t, true, false)
			err := p.attempt(request.ProtocolConfig{CACert: missing}, request.Request{URL: p.url(srv), InsecureTLS: true})
			if err == nil || !strings.Contains(err.Error(), "CA file") {
				t.Fatalf("want an error naming the CA file, got %v", err)
			}
		})
	}
}

func TestSSEAndMCPUseConfiguredProxy(t *testing.T) {
	for _, p := range []httpProtocol{sseProtocol, mcpProtocol} {
		t.Run(p.name, func(t *testing.T) {
			proxy, log := p.serve(t, false, false)
			err := p.attempt(request.ProtocolConfig{Proxy: proxy.URL}, request.Request{URL: "http://target.invalid/events"})
			if err != nil {
				t.Fatal(err)
			}
			if got := log.first(t).Host; got != "target.invalid" {
				t.Fatalf("the proxy saw a request for host %q", got)
			}
		})
	}
}

func TestHTTPProtocolsSendEveryEnabledHeader(t *testing.T) {
	headers := []request.KV{
		{Name: "X-Multi", Value: "one", Enabled: true},
		{Name: "X-Multi", Value: "two", Enabled: true},
		{Name: "X-Off", Value: "no", Enabled: false},
		{Name: "", Value: "unnamed", Enabled: true},
	}
	for _, p := range []httpProtocol{restProtocol, sseProtocol, mcpProtocol} {
		t.Run(p.name, func(t *testing.T) {
			srv, log := p.serve(t, false, false)
			if err := p.attempt(request.ProtocolConfig{}, request.Request{URL: p.url(srv), Headers: headers}); err != nil {
				t.Fatal(err)
			}
			header := log.first(t).Header
			if got := header.Values("X-Multi"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
				t.Errorf("X-Multi = %v, want both rows", got)
			}
			if header.Get("X-Off") != "" {
				t.Errorf("a disabled header was sent: %v", header)
			}
		})
	}
}

func TestSSEUserAcceptHeaderOverridesDefault(t *testing.T) {
	srv, log := sseProtocol.serve(t, false, false)
	call := request.Request{URL: srv.URL, Headers: []request.KV{{Name: "Accept", Value: "text/plain", Enabled: true}}}
	if err := sseProtocol.attempt(request.ProtocolConfig{}, call); err != nil {
		t.Fatal(err)
	}
	if got := log.first(t).Header.Values("Accept"); len(got) != 1 || got[0] != "text/plain" {
		t.Fatalf("Accept = %v", got)
	}
}

func TestAuthPaneWinsOverTypedHeader(t *testing.T) {
	cases := []struct {
		name    string
		header  request.KV
		auth    request.Auth
		wantKey string
		want    string
	}{
		{"bearer", request.KV{Name: "Authorization", Value: "Bearer typed", Enabled: true}, request.Auth{Type: request.AuthBearer, Token: "pane"}, "Authorization", "Bearer pane"},
		{"api key header", request.KV{Name: "X-Key", Value: "typed", Enabled: true}, request.Auth{Type: request.AuthAPIKey, KeyName: "X-Key", KeyValue: "pane"}, "X-Key", "pane"},
	}
	for _, p := range allFour {
		for _, c := range cases {
			t.Run(p.name+" "+c.name, func(t *testing.T) {
				srv, log := p.serve(t, false, false)
				call := request.Request{URL: p.url(srv), Headers: []request.KV{c.header}, Auth: c.auth}
				if err := p.attempt(request.ProtocolConfig{}, call); err != nil {
					t.Fatal(err)
				}
				if got := log.first(t).Header.Values(c.wantKey); len(got) != 1 || got[0] != c.want {
					t.Fatalf("%s = %v, want only %q", c.wantKey, got, c.want)
				}
			})
		}
	}
}

func TestHTTPProtocolsRejectUnknownAuthType(t *testing.T) {
	for _, p := range allFour {
		t.Run(p.name, func(t *testing.T) {
			srv, log := p.serve(t, false, false)
			err := p.attempt(request.ProtocolConfig{}, request.Request{URL: p.url(srv), Auth: request.Auth{Type: "nope"}})
			if err == nil || !strings.Contains(err.Error(), `unknown auth type "nope"`) {
				t.Fatalf("got %v", err)
			}
			if got := len(log.all()); got != 0 {
				t.Fatalf("the server saw %d requests", got)
			}
		})
	}
}

// The API key in the query was dropped: the handshake built its headers
// on a probe request and dialed the URL as typed.
func TestWebSocketSendsAPIKeyInQuery(t *testing.T) {
	srv, log := wsProtocol.serve(t, false, false)
	call := request.Request{URL: wsProtocol.url(srv) + "/chat?room=1", Auth: request.Auth{Type: request.AuthAPIKey, KeyName: "key", KeyValue: "secret", KeyIn: request.KeyInQuery}}
	if err := wsProtocol.attempt(request.ProtocolConfig{}, call); err != nil {
		t.Fatal(err)
	}
	query, _ := url.ParseQuery(log.first(t).RawQuery)
	if query.Get("key") != "secret" || query.Get("room") != "1" {
		t.Fatalf("query %q", log.first(t).RawQuery)
	}
}

func TestSSEAndMCPSendConfiguredCookies(t *testing.T) {
	for _, p := range []httpProtocol{sseProtocol, mcpProtocol} {
		t.Run(p.name, func(t *testing.T) {
			srv, log := p.serve(t, false, false)
			jar, _ := cookiejar.New(nil)
			address, _ := url.Parse(srv.URL)
			jar.SetCookies(address, []*http.Cookie{{Name: "session", Value: "abc", Path: "/"}})
			if err := p.attempt(request.ProtocolConfig{Jar: jar}, request.Request{URL: srv.URL}); err != nil {
				t.Fatal(err)
			}
			if got := log.first(t).Header.Get("Cookie"); got != "session=abc" {
				t.Fatalf("Cookie = %q", got)
			}
		})
	}
}

func TestNewHTTPRequestBuildsBodyHeadersAndAuthInOrder(t *testing.T) {
	cases := []struct {
		name        string
		request     request.Request
		contentType string
		check       func(t *testing.T, built *http.Request)
	}{
		{"automatic content type", request.Request{URL: "http://x/"}, "application/json", func(t *testing.T, built *http.Request) {
			if got := built.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q", got)
			}
		}},
		{"no content type when empty", request.Request{URL: "http://x/"}, "", func(t *testing.T, built *http.Request) {
			if _, ok := built.Header["Content-Type"]; ok {
				t.Errorf("headers %v", built.Header)
			}
		}},
		{"typed content type replaces the automatic one", request.Request{URL: "http://x/", Headers: []request.KV{{Name: "content-type", Value: "text/csv", Enabled: true}}}, "application/json", func(t *testing.T, built *http.Request) {
			if got := built.Header.Values("Content-Type"); len(got) != 1 || got[0] != "text/csv" {
				t.Errorf("Content-Type = %v", got)
			}
		}},
		{"auth pane wins over a typed header", request.Request{URL: "http://x/", Headers: []request.KV{{Name: "Authorization", Value: "typed", Enabled: true}}, Auth: request.Auth{Type: request.AuthBearer, Token: "pane"}}, "", func(t *testing.T, built *http.Request) {
			if got := built.Header.Values("Authorization"); len(got) != 1 || got[0] != "Bearer pane" {
				t.Errorf("Authorization = %v", got)
			}
		}},
		{"api key in the query", request.Request{URL: "http://x/?a=1", Auth: request.Auth{Type: request.AuthAPIKey, KeyName: "k", KeyValue: "v", KeyIn: request.KeyInQuery}}, "", func(t *testing.T, built *http.Request) {
			if got := built.URL.Query(); got.Get("k") != "v" || got.Get("a") != "1" {
				t.Errorf("URL %s", built.URL)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			built, err := newHTTPRequest(context.Background(), http.MethodPost, c.request, []byte("body"), c.contentType)
			if err != nil {
				t.Fatal(err)
			}
			if body, _ := io.ReadAll(built.Body); string(body) != "body" || built.Method != http.MethodPost {
				t.Errorf("%s %q", built.Method, body)
			}
			c.check(t, built)
		})
	}
	if _, err := newHTTPRequest(context.Background(), http.MethodGet, request.Request{URL: "http://x/", Auth: request.Auth{Type: "nope"}}, nil, ""); err == nil || err.Error() != `unknown auth type "nope"` {
		t.Errorf("unknown auth: %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(call *http.Request) (*http.Response, error) { return f(call) }

func TestNewHTTPClientAppliesNetworkAndKeepsTheBase(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	other, _ := cookiejar.New(nil)
	custom := roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, nil })

	client, err := newHTTPClient(nil, Network{Jar: jar}, request.Request{})
	if err != nil || client.Jar != jar || client.Transport != nil {
		t.Errorf("plain client with a jar: %+v, %v", client, err)
	}
	base := &http.Client{Jar: other}
	client, _ = newHTTPClient(base, Network{Jar: jar}, request.Request{})
	if client.Jar != other || client == base {
		t.Error("the base client's own jar wins and the base is copied")
	}
	client, _ = newHTTPClient(nil, Network{}, request.Request{NoRedirects: true})
	if client.CheckRedirect == nil {
		t.Error("NoRedirects must stop redirects")
	}
	client, _ = newHTTPClient(&http.Client{Transport: custom}, Network{Proxy: "http://p:1"}, request.Request{InsecureTLS: true})
	if _, ok := client.Transport.(roundTripperFunc); !ok {
		t.Errorf("a custom round tripper is left alone, got %T", client.Transport)
	}
	client, err = newHTTPClient(nil, Network{Proxy: "http://p:1"}, request.Request{InsecureTLS: true})
	transport, ok := client.Transport.(*http.Transport)
	if err != nil || !ok || transport.Proxy == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Errorf("proxy and insecure TLS: %+v, %v", client.Transport, err)
	}
	if _, err := newHTTPClient(nil, Network{Proxy: "ftp://p:21"}, request.Request{}); err == nil || !strings.Contains(err.Error(), "not a URL like http://host:port") {
		t.Errorf("bad proxy: %v", err)
	}
}
