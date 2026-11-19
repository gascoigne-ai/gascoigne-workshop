package model

import (
	"fmt"
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

// Clustered short flags (-sSL, -XPOST, -uuser:pw, -d'x') fell into the
// default branch and were ignored, so the method, -L and the auth were
// lost with no error, and an unknown --flag=value turned its value into
// the URL.
func TestParseCurlReadsClusteredAndAttachedFlags(t *testing.T) {
	tests := []struct {
		name, cmd string
		check     func(Request) string
	}{
		{"attached method", `curl -XPOST https://api.test`, func(request Request) string { return request.Method }},
		{"cluster with follow", `curl -sSL https://api.test`, func(request Request) string { return fmt.Sprint(request.NoRedirects) }},
		{"attached user", `curl -ualice:pw https://api.test`, func(request Request) string { return request.Auth.User + "/" + request.Auth.Password }},
		{"attached data", `curl -d'a=1' https://api.test`, func(request Request) string {
			return request.Method + " " + request.Form[0].Name + "=" + request.Form[0].Value
		}},
		{"cluster then value flag", `curl -sXDELETE https://api.test`, func(request Request) string { return request.Method }},
		{"unknown long flag with a value", `curl --some-flag=https://evil.test https://api.test`, func(request Request) string { return request.URL }},
		{"known long flag with =", `curl --request=PUT https://api.test`, func(request Request) string { return request.Method }},
	}
	want := map[string]string{
		"attached method":                "POST",
		"cluster with follow":            "false",
		"attached user":                  "alice/pw",
		"attached data":                  "POST a=1",
		"cluster then value flag":        "DELETE",
		"unknown long flag with a value": "https://api.test",
		"known long flag with =":         "PUT",
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurl(tt.cmd)
			if err != nil {
				t.Fatal(err)
			}
			if result := tt.check(got); result != want[tt.name] {
				t.Fatalf("got %q, want %q", result, want[tt.name])
			}
		})
	}
}

func TestParseCurlKeepsAValueThatLooksLikeAFlag(t *testing.T) {
	got, err := ParseCurl(`curl -d '-x=1' -H '-odd: v' https://api.test`)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://api.test" || len(got.Form) != 1 || got.Form[0].Name != "-x" || got.Headers[0].Name != "-odd" {
		t.Fatalf("%+v", got)
	}
}

// FuzzParseCurl: any command line may be refused, none may panic.
func FuzzParseCurl(fuzz *testing.F) {
	for _, seed := range []string{
		`curl https://api.test/users`, `curl -X POST 'https://api.test/u' -H 'Content-Type: application/json' --data-raw '{"a":1}'`,
		`curl -sSL -u alice:pw https://api.test`, `curl -XPOST -d'a=1' https://api.test`, `curl -F 'file=@a.txt' https://api.test/up`,
		"curl \\\n  --url https://api.test \\\n  --request DELETE", `curl https://api.test -d $'{"a":"x\ny"}'`, `curl '`, ``,
	} {
		fuzz.Add(seed)
	}
	fuzz.Fuzz(func(t *testing.T, command string) {
		_, _ = ParseCurl(command)
	})
}

func TestImportedNameIsMethodAndPath(t *testing.T) {
	tests := []struct {
		name, method, url, want string
	}{
		{"path", "GET", "https://api.example.com/users/1", "GET /users/1"},
		{"query dropped", "POST", "https://api.example.com/users?page=2", "POST /users"},
		{"no path", "GET", "https://api.example.com", "GET /"},
		{"no path with query", "GET", "https://api.example.com?x=1", "GET /"},
		{"no scheme", "DELETE", "localhost:8080/a/b", "DELETE /a/b"},
		{"variable host", "PUT", "{{baseUrl}}/items", "PUT /items"},
		{"empty", "GET", "", "GET /"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ImportedName(Request{Method: test.method, URL: test.url}); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
