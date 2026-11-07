package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCurl(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want Request
	}{
		{"plain get", `curl https://api.test/users`,
			Request{Protocol: "rest", Method: "GET", URL: "https://api.test/users", BodyType: BodyNone}},
		{"query becomes params", `curl 'https://api.test/s?q=go+lang&page=2'`,
			Request{Protocol: "rest", Method: "GET", URL: "https://api.test/s?q=go+lang&page=2", BodyType: BodyNone,
				Params: []KV{{Name: "q", Value: "go lang", Enabled: true}, {Name: "page", Value: "2", Enabled: true}}}},
		{"json post with headers", `curl -X POST 'https://api.test/u' -H 'Content-Type: application/json' -H "Accept: */*" --data-raw '{"a":1}'`,
			Request{Protocol: "rest", Method: "POST", URL: "https://api.test/u", BodyType: BodyJSON, Body: `{"a":1}`,
				Headers: []KV{{Name: "Content-Type", Value: "application/json", Enabled: true}, {Name: "Accept", Value: "*/*", Enabled: true}}}},
		{"data implies post", `curl https://api.test/u -d 'x=1&y=a%20b'`,
			Request{Protocol: "rest", Method: "POST", URL: "https://api.test/u", BodyType: BodyForm,
				Form: []KV{{Name: "x", Value: "1", Enabled: true}, {Name: "y", Value: "a b", Enabled: true}}}},
		{"bearer token", `curl https://api.test -H 'Authorization: Bearer abc.def'`,
			Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: BodyNone,
				Auth: Auth{Type: AuthBearer, Token: "abc.def"}}},
		{"basic auth", `curl -u alice:pw https://api.test`,
			Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: BodyNone,
				Auth: Auth{Type: AuthBasic, User: "alice", Password: "pw"}}},
		{"line continuations", "curl \\\n  --url https://api.test \\\n  --request DELETE",
			Request{Protocol: "rest", Method: "DELETE", URL: "https://api.test", BodyType: BodyNone}},
		{"follow redirects flag", `curl -L -s --compressed -k https://api.test`,
			Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: BodyNone}},
		{"multipart", `curl -F 'name=x' -F "file=@a.txt" https://api.test/up`,
			Request{Protocol: "rest", Method: "POST", URL: "https://api.test/up", BodyType: BodyMultipart,
				Form: []KV{{Name: "name", Value: "x", Enabled: true}, {Name: "file", Value: "@a.txt", Enabled: true}}}},
		{"ansi-c quoting", `curl https://api.test -d $'{"a":"x\ny"}' -H 'Content-Type: application/json'`,
			Request{Protocol: "rest", Method: "POST", URL: "https://api.test", BodyType: BodyJSON, Body: "{\"a\":\"x\ny\"}",
				Headers: []KV{{Name: "Content-Type", Value: "application/json", Enabled: true}}}},
		{"head", `curl -I https://api.test`,
			Request{Protocol: "rest", Method: "HEAD", URL: "https://api.test", BodyType: BodyNone}},
		{"cookie and agent", `curl https://api.test -b 'a=1' -A 'me/1'`,
			Request{Protocol: "rest", Method: "GET", URL: "https://api.test", BodyType: BodyNone,
				Headers: []KV{{Name: "Cookie", Value: "a=1", Enabled: true}, {Name: "User-Agent", Value: "me/1", Enabled: true}}}},
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
