package proto

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestParseCurl(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want model.Request
	}{
		{"plain get", `curl https://api.test/users`,
			model.Request{Protocol: "rest", Method: "GET", URL: "https://api.test/users", BodyType: model.BodyNone}},
		{"query becomes params", `curl 'https://api.test/s?q=go+lang&page=2'`,
			model.Request{Protocol: "rest", Method: "GET", URL: "https://api.test/s?q=go+lang&page=2", BodyType: model.BodyNone,
				Params: []model.KV{{Name: "q", Value: "go lang", Enabled: true}, {Name: "page", Value: "2", Enabled: true}}}},
		{"json post with headers", `curl -X POST 'https://api.test/u' -H 'Content-Type: application/json' -H "Accept: */*" --data-raw '{"a":1}'`,
			model.Request{Protocol: "rest", Method: "POST", URL: "https://api.test/u", BodyType: model.BodyJSON, Body: `{"a":1}`,
				Headers: []model.KV{{Name: "Content-Type", Value: "application/json", Enabled: true}, {Name: "Accept", Value: "*/*", Enabled: true}}}},
		{"data implies post", `curl https://api.test/u -d 'x=1&y=a%20b'`,
			model.Request{Protocol: "rest", Method: "POST", URL: "https://api.test/u", BodyType: model.BodyForm,
				Form: []model.KV{{Name: "x", Value: "1", Enabled: true}, {Name: "y", Value: "a b", Enabled: true}}}},
		{"bearer token", `curl https://api.test -H 'Authorization: Bearer abc.def'`,
			model.Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: model.BodyNone,
				Auth: model.Auth{Type: model.AuthBearer, Token: "abc.def"}}},
		{"basic auth", `curl -u alice:pw https://api.test`,
			model.Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: model.BodyNone,
				Auth: model.Auth{Type: model.AuthBasic, User: "alice", Password: "pw"}}},
		{"line continuations", "curl \\\n  --url https://api.test \\\n  --request DELETE",
			model.Request{Protocol: "rest", Method: "DELETE", URL: "https://api.test", BodyType: model.BodyNone}},
		{"follow redirects flag", `curl -L -s --compressed -k https://api.test`,
			model.Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: model.BodyNone}},
		{"multipart", `curl -F 'name=x' -F "file=@a.txt" https://api.test/up`,
			model.Request{Protocol: "rest", Method: "POST", URL: "https://api.test/up", BodyType: model.BodyMultipart,
				Form: []model.KV{{Name: "name", Value: "x", Enabled: true}, {Name: "file", Value: "@a.txt", Enabled: true}}}},
		{"ansi-c quoting", `curl https://api.test -d $'{"a":"x\ny"}' -H 'Content-Type: application/json'`,
			model.Request{Protocol: "rest", Method: "POST", URL: "https://api.test", BodyType: model.BodyJSON, Body: "{\"a\":\"x\ny\"}",
				Headers: []model.KV{{Name: "Content-Type", Value: "application/json", Enabled: true}}}},
		{"head", `curl -I https://api.test`,
			model.Request{Protocol: "rest", Method: "HEAD", URL: "https://api.test", BodyType: model.BodyNone}},
		{"cookie and agent", `curl https://api.test -b 'a=1' -A 'me/1'`,
			model.Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: model.BodyNone,
				Headers: []model.KV{{Name: "Cookie", Value: "a=1", Enabled: true}, {Name: "User-Agent", Value: "me/1", Enabled: true}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurl(tt.cmd)
			if err != nil {
				t.Fatal(err)
			}
			tt.want.NoRedirects = !strings.Contains(tt.cmd, "-L") // curl follows redirects only with -L
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestParseCurlErrors(t *testing.T) {
	for cmd, want := range map[string]string{
		"":                            "empty",
		"wget https://x":              "not a curl command",
		"curl -X":                     "needs a value",
		"curl -H 'a: b'":              "no URL",
		`curl 'https://x -H a`:        "unterminated",
		"curl --weird-flag https://x": "", // unknown flags are skipped
	} {
		_, err := ParseCurl(cmd)
		if want == "" {
			if err != nil {
				t.Errorf("%q: %v", cmd, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want %q", cmd, err, want)
		}
	}
}

func TestCurlRoundTrip(t *testing.T) {
	req := model.Request{Protocol: "rest", Method: "POST", URL: "https://api.test/u", BodyType: model.BodyJSON, Body: `{"it's":"ok"}`,
		Headers: []model.KV{{Name: "X-A", Value: "1", Enabled: true}}, Auth: model.Auth{Type: model.AuthBearer, Token: "t"}}
	got, err := ParseCurl(Curl(req))
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != req.Body || got.Method != "POST" || got.URL != req.URL || got.Auth.Token != "t" {
		t.Fatalf("%+v", got)
	}
}
