package proto

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestCurl(t *testing.T) {
	tests := []struct {
		name string
		req  model.Request
		want string
	}{
		{"plain get", model.Request{Method: "GET", URL: "https://x.test/a?b=1"},
			`curl -L 'https://x.test/a?b=1'`},
		{"empty method is get", model.Request{URL: "https://x.test"}, `curl -L 'https://x.test'`},
		{"other methods are named", model.Request{Method: "DELETE", URL: "https://x.test"},
			`curl -L -X DELETE 'https://x.test'`},
		{"redirects are not followed when switched off", model.Request{URL: "https://x.test", NoRedirects: true},
			`curl 'https://x.test'`},
		{"enabled headers only, in order", model.Request{URL: "https://x.test", Headers: []model.KV{
			{Name: "Accept", Value: "text/html", Enabled: true},
			{Name: "X-Off", Value: "no"},
			{Name: "", Value: "unnamed", Enabled: true},
			{Name: "X-B", Value: "2", Enabled: true},
		}}, `curl -L 'https://x.test' -H 'Accept: text/html' -H 'X-B: 2'`},
		{"json body", model.Request{Method: "POST", URL: "https://x.test", BodyType: model.BodyJSON, Body: `{"a":"it's"}`},
			`curl -L -X POST 'https://x.test' -H 'Content-Type: application/json' --data-raw '{"a":"it'\''s"}'`},
		{"text body", model.Request{Method: "POST", URL: "https://x.test", BodyType: model.BodyText, Body: "hi"},
			`curl -L -X POST 'https://x.test' -H 'Content-Type: text/plain; charset=utf-8' --data-raw 'hi'`},
		{"none ignores the body", model.Request{Method: "POST", URL: "https://x.test", BodyType: model.BodyNone, Body: "x"},
			`curl -L -X POST 'https://x.test'`},
		{"form fields", model.Request{Method: "POST", URL: "https://x.test", BodyType: model.BodyForm, Form: []model.KV{
			{Name: "q", Value: "a b", Enabled: true}, {Name: "off", Value: "x"},
		}}, `curl -L -X POST 'https://x.test' --data-urlencode 'q=a b'`},
		{"multipart fields", model.Request{Method: "POST", URL: "https://x.test", BodyType: model.BodyMultipart, Form: []model.KV{
			{Name: "a", Value: "1", Enabled: true},
		}}, `curl -L -X POST 'https://x.test' -F 'a=1'`},
		{"user content type replaces the automatic one", model.Request{Method: "POST", URL: "https://x.test", BodyType: model.BodyJSON, Body: "{}",
			Headers: []model.KV{{Name: "content-type", Value: "application/vnd.api+json", Enabled: true}}},
			`curl -L -X POST 'https://x.test' -H 'content-type: application/vnd.api+json' --data-raw '{}'`},
		{"bearer", model.Request{URL: "https://x.test", Auth: model.Auth{Type: model.AuthBearer, Token: "abc"}},
			`curl -L 'https://x.test' -H 'Authorization: Bearer abc'`},
		{"basic", model.Request{URL: "https://x.test", Auth: model.Auth{Type: model.AuthBasic, User: "ann", Password: "p w"}},
			`curl -L 'https://x.test' -u 'ann:p w'`},
		{"api key header", model.Request{URL: "https://x.test", Auth: model.Auth{Type: model.AuthAPIKey, KeyName: "X-Key", KeyValue: "v"}},
			`curl -L 'https://x.test' -H 'X-Key: v'`},
		{"api key query joins the url", model.Request{URL: "https://x.test/p?x=1", Auth: model.Auth{Type: model.AuthAPIKey, KeyIn: model.KeyInQuery, KeyName: "key", KeyValue: "a b"}},
			`curl -L 'https://x.test/p?key=a+b&x=1'`},
		{"quotes in values survive", model.Request{URL: "https://x.test/it's"}, `curl -L 'https://x.test/it'\''s'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Curl(tt.req); got != tt.want {
				t.Fatalf("\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}
