package proto

import (
	"testing"

	request "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
)

func TestCurl(t *testing.T) {
	tests := []struct {
		name string
		req  request.Request
		want string
	}{
		{"plain get", request.Request{Method: "GET", URL: "https://x.test/a?b=1"},
			`curl -L 'https://x.test/a?b=1'`},
		{"empty method is get", request.Request{URL: "https://x.test"}, `curl -L 'https://x.test'`},
		{"other methods are named", request.Request{Method: "DELETE", URL: "https://x.test"},
			`curl -L -X DELETE 'https://x.test'`},
		// The method went to the command line unquoted, so a method from an
		// imported file such as "GET; rm x" ran as a shell command when pasted.
		{"a method with shell characters is quoted", request.Request{Method: "get; rm x", URL: "https://x.test"},
			`curl -L -X 'GET; RM X' 'https://x.test'`},
		{"redirects are not followed when switched off", request.Request{URL: "https://x.test", NoRedirects: true},
			`curl 'https://x.test'`},
		{"enabled headers only, in order", request.Request{URL: "https://x.test", Headers: []request.KV{
			{Name: "Accept", Value: "text/html", Enabled: true},
			{Name: "X-Off", Value: "no"},
			{Name: "", Value: "unnamed", Enabled: true},
			{Name: "X-B", Value: "2", Enabled: true},
		}}, `curl -L 'https://x.test' -H 'Accept: text/html' -H 'X-B: 2'`},
		{"json body", request.Request{Method: "POST", URL: "https://x.test", BodyType: request.BodyJSON, Body: `{"a":"it's"}`},
			`curl -L -X POST 'https://x.test' -H 'Content-Type: application/json' --data-raw '{"a":"it'\''s"}'`},
		{"text body", request.Request{Method: "POST", URL: "https://x.test", BodyType: request.BodyText, Body: "hi"},
			`curl -L -X POST 'https://x.test' -H 'Content-Type: text/plain; charset=utf-8' --data-raw 'hi'`},
		{"none ignores the body", request.Request{Method: "POST", URL: "https://x.test", BodyType: request.BodyNone, Body: "x"},
			`curl -L -X POST 'https://x.test'`},
		{"form fields", request.Request{Method: "POST", URL: "https://x.test", BodyType: request.BodyForm, Form: []request.KV{
			{Name: "q", Value: "a b", Enabled: true}, {Name: "off", Value: "x"},
		}}, `curl -L -X POST 'https://x.test' --data-urlencode 'q=a b'`},
		{"multipart fields", request.Request{Method: "POST", URL: "https://x.test", BodyType: request.BodyMultipart, Form: []request.KV{
			{Name: "a", Value: "1", Enabled: true},
		}}, `curl -L -X POST 'https://x.test' -F 'a=1'`},
		{"user content type replaces the automatic one", request.Request{Method: "POST", URL: "https://x.test", BodyType: request.BodyJSON, Body: "{}",
			Headers: []request.KV{{Name: "content-type", Value: "application/vnd.api+json", Enabled: true}}},
			`curl -L -X POST 'https://x.test' -H 'content-type: application/vnd.api+json' --data-raw '{}'`},
		{"bearer", request.Request{URL: "https://x.test", Auth: request.Auth{Type: request.AuthBearer, Token: "abc"}},
			`curl -L 'https://x.test' -H 'Authorization: Bearer abc'`},
		{"basic", request.Request{URL: "https://x.test", Auth: request.Auth{Type: request.AuthBasic, User: "ann", Password: "p w"}},
			`curl -L 'https://x.test' -u 'ann:p w'`},
		{"api key header", request.Request{URL: "https://x.test", Auth: request.Auth{Type: request.AuthAPIKey, KeyName: "X-Key", KeyValue: "v"}},
			`curl -L 'https://x.test' -H 'X-Key: v'`},
		{"api key query joins the url", request.Request{URL: "https://x.test/p?x=1", Auth: request.Auth{Type: request.AuthAPIKey, KeyIn: request.KeyInQuery, KeyName: "key", KeyValue: "a b"}},
			`curl -L 'https://x.test/p?key=a+b&x=1'`},
		{"quotes in values survive", request.Request{URL: "https://x.test/it's"}, `curl -L 'https://x.test/it'\''s'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Curl(tt.req); got != tt.want {
				t.Fatalf("\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}

// Curl and ParseCurl read each other.
func TestCurlRoundTrip(t *testing.T) {
	req := request.Request{Protocol: "rest", Method: "POST", URL: "https://api.test/u", BodyType: request.BodyJSON, Body: `{"it's":"ok"}`,
		Headers: []request.KV{{Name: "X-A", Value: "1", Enabled: true}}, Auth: request.Auth{Type: request.AuthBearer, Token: "t"}}
	got, err := send.ParseCurl(Curl(req))
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != req.Body || got.Method != "POST" || got.URL != req.URL || got.Auth.Token != "t" {
		t.Fatalf("%+v", got)
	}
}
