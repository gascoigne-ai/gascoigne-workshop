package proto

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestRegistryNames(t *testing.T) {
	want := []string{"grpc", "mcp", "rest", "sse", "websocket"}
	if got := NewRegistry().Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestRegistrySend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	tests := []struct {
		name    string
		req     model.Request
		wantErr bool
		wantMsg string // substring of Response.Err
		want    int    // Response.Status
	}{
		{"rest", model.Request{Protocol: "rest", URL: srv.URL}, false, "", 200},
		{"websocket", model.Request{Protocol: "websocket"}, false, "URL scheme", 0},
		{"mcp", model.Request{Protocol: "mcp"}, false, "not an http or https URL", 0},
		{"unknown", model.Request{Protocol: "carrier-pigeon"}, true, "", 0},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Echo-Method", r.Method)
		w.Header().Set("X-Echo-Auth", r.Header.Get("Authorization"))
		w.Header().Set("X-Echo-Skipped", r.Header.Get("X-Off"))
		w.WriteHeader(http.StatusCreated)
		w.Write(body)
	}))
	defer srv.Close()

	resp, err := REST{}.Send(context.Background(), model.Request{
		Method: "post",
		URL:    srv.URL,
		Body:   `{"a":1}`,
		Headers: []model.KV{
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
		if _, err := (REST{}).Send(context.Background(), model.Request{Method: "BAD METHOD", URL: "http://x"}); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("connection refused lands in Err", func(t *testing.T) {
		resp, err := REST{}.Send(context.Background(), model.Request{URL: closed.URL})
		if err != nil || resp.Err == "" || resp.Status != 0 {
			t.Fatalf("resp = %+v err = %v", resp, err)
		}
	})
	t.Run("cancelled context lands in Err", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		resp, err := REST{}.Send(ctx, model.Request{URL: closed.URL})
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.contentType = r.Method, r.Header.Get("Content-Type")
		got.body, _ = io.ReadAll(r.Body)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func TestRESTBodyTypes(t *testing.T) {
	form := []model.KV{
		{Name: "q", Value: "a b&c", Enabled: true},
		{Name: "off", Value: "x", Enabled: false},
		{Name: "", Value: "unnamed", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
	}
	tests := []struct {
		name            string
		req             model.Request
		wantBody        string
		wantContentType string
	}{
		{"none sends nothing", model.Request{BodyType: model.BodyNone, Body: "ignored"}, "", ""},
		{"legacy raw body has no content type", model.Request{Body: `{"a":1}`}, `{"a":1}`, ""},
		{"json", model.Request{BodyType: model.BodyJSON, Body: `{"a":1}`}, `{"a":1}`, "application/json"},
		{"text", model.Request{BodyType: model.BodyText, Body: "hello"}, "hello", "text/plain; charset=utf-8"},
		{"form skips disabled and unnamed rows and escapes", model.Request{BodyType: model.BodyForm, Form: form}, "q=a+b%26c&page=2", "application/x-www-form-urlencoded"},
		{"form ignores the text body", model.Request{BodyType: model.BodyForm, Body: "text", Form: form[:1]}, "q=a+b%26c", "application/x-www-form-urlencoded"},
		{"json body ignores form rows", model.Request{BodyType: model.BodyJSON, Body: "{}", Form: form}, "{}", "application/json"},
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
	_, err := REST{}.Send(context.Background(), model.Request{
		Method: "POST", URL: srv.URL, BodyType: model.BodyMultipart,
		Form: []model.KV{{Name: "a", Value: "1", Enabled: true}, {Name: "skip", Value: "x"}, {Name: "b", Value: "héllo", Enabled: true}},
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
		headers []model.KV
		want    string
	}{
		{"user header replaces the automatic one", []model.KV{{Name: "content-type", Value: "application/vnd.api+json", Enabled: true}}, "application/vnd.api+json"},
		{"a disabled header does not", []model.KV{{Name: "Content-Type", Value: "x/y", Enabled: false}}, "application/json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, got := captureServer(t)
			REST{}.Send(context.Background(), model.Request{Method: "POST", URL: srv.URL, BodyType: model.BodyJSON, Body: "{}", Headers: tt.headers})
			if got.contentType != tt.want {
				t.Fatalf("content type = %q, want %q", got.contentType, tt.want)
			}
		})
	}
}

func TestRESTRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "landed") }))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
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
			resp, err := REST{}.Send(context.Background(), model.Request{URL: redirector.URL, NoRedirects: tt.noRedirects})
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, _ := REST{}.Send(ctx, model.Request{URL: srv.URL})
	if resp.Err != ErrCancelled {
		t.Errorf("cancelled: Err = %q, want %q", resp.Err, ErrCancelled)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	resp, _ = REST{}.Send(ctx, model.Request{URL: srv.URL})
	if resp.Err != ErrTimedOut {
		t.Errorf("timeout: Err = %q, want %q", resp.Err, ErrTimedOut)
	}
}

func TestRESTAuthTypes(t *testing.T) {
	tests := []struct {
		name       string
		auth       model.Auth
		path       string
		wantHeader [2]string // name, value that must arrive ("" name: none)
		wantQuery  string    // raw query that must arrive
	}{
		{"none", model.Auth{}, "/p", [2]string{}, ""},
		{"explicit none", model.Auth{Type: model.AuthNone, Token: "ignored"}, "/p", [2]string{}, ""},
		{"bearer", model.Auth{Type: model.AuthBearer, Token: "abc"}, "/p", [2]string{"Authorization", "Bearer abc"}, ""},
		{"basic", model.Auth{Type: model.AuthBasic, User: "ann", Password: "pw:x"}, "/p", [2]string{"Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte("ann:pw:x"))}, ""},
		{"api key header", model.Auth{Type: model.AuthAPIKey, KeyName: "X-Api-Key", KeyValue: "k1"}, "/p", [2]string{"X-Api-Key", "k1"}, ""},
		{"api key header is the default place", model.Auth{Type: model.AuthAPIKey, KeyIn: model.KeyInHeader, KeyName: "X-Api-Key", KeyValue: "k1"}, "/p", [2]string{"X-Api-Key", "k1"}, ""},
		{"api key in the query", model.Auth{Type: model.AuthAPIKey, KeyIn: model.KeyInQuery, KeyName: "key", KeyValue: "a b"}, "/p", [2]string{}, "key=a+b"},
		{"api key keeps the existing query", model.Auth{Type: model.AuthAPIKey, KeyIn: model.KeyInQuery, KeyName: "key", KeyValue: "k"}, "/p?x=1", [2]string{}, "key=k&x=1"},
		{"api key without a name adds nothing", model.Auth{Type: model.AuthAPIKey, KeyValue: "k"}, "/p", [2]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotHeader http.Header
			var gotQuery string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotHeader, gotQuery = r.Header.Clone(), r.URL.RawQuery
			}))
			defer srv.Close()
			_, err := REST{}.Send(context.Background(), model.Request{URL: srv.URL + tt.path, Auth: tt.auth})
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.Header.Get("Authorization") }))
	defer srv.Close()
	REST{}.Send(context.Background(), model.Request{
		URL:     srv.URL,
		Headers: []model.KV{{Name: "Authorization", Value: "manual", Enabled: true}},
		Auth:    model.Auth{Type: model.AuthBearer, Token: "t"},
	})
	if got != "Bearer t" {
		t.Fatalf("Authorization = %q: the Auth pane wins over a header typed by hand", got)
	}
}

func TestUnknownAuthTypeIsAnError(t *testing.T) {
	if _, err := (REST{}).Send(context.Background(), model.Request{URL: "http://x", Auth: model.Auth{Type: "kerberos"}}); err == nil {
		t.Fatal("want an error for an unknown auth type")
	}
}
