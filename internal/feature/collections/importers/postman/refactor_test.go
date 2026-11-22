package postman

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestImportListsARequestWhoseURLCannotBeRead(t *testing.T) {
	data := `{"info":{"name":"C","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
	"item":[{"name":"Broken","request":{"method":"GET","url":5}}]}`
	res, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || !strings.HasPrefix(res.Skipped[0], "Broken: ") || !strings.Contains(res.Skipped[0], "url") {
		t.Fatalf("Skipped = %q, want one entry naming the request and the url", res.Skipped)
	}
}

// Pins the exported shape of every body mode and auth type, so the shared
// tables cannot change what a file contains.
func TestExportWritesEveryBodyModeAndAuthType(t *testing.T) {
	cases := []struct {
		name    string
		request request.Request
		want    []string
	}{
		{"json body", request.Request{Protocol: request.ProtocolREST, Method: "POST", BodyType: request.BodyJSON, Body: "{}"},
			[]string{`"mode": "raw"`, `"language": "json"`}},
		{"text body", request.Request{Protocol: request.ProtocolREST, Method: "POST", BodyType: request.BodyText, Body: "x"},
			[]string{`"mode": "raw"`, `"raw": "x"`}},
		{"form body", request.Request{Protocol: request.ProtocolREST, Method: "POST", BodyType: request.BodyForm, Form: []request.KV{{Name: "a", Value: "1", Enabled: true}}},
			[]string{`"mode": "urlencoded"`, `"key": "a"`}},
		{"multipart body", request.Request{Protocol: request.ProtocolREST, Method: "POST", BodyType: request.BodyMultipart, Form: []request.KV{{Name: "f", Value: "@/tmp/x", Enabled: true}}},
			[]string{`"mode": "formdata"`, `"type": "file"`, `"src": "/tmp/x"`}},
		{"bearer", request.Request{Protocol: request.ProtocolREST, Method: "GET", Auth: request.Auth{Type: request.AuthBearer, Token: "{{t}}"}},
			[]string{`"type": "bearer"`, `"key": "token"`, `"value": "{{t}}"`}},
		{"basic", request.Request{Protocol: request.ProtocolREST, Method: "GET", Auth: request.Auth{Type: request.AuthBasic, User: "u", Password: "plain"}},
			[]string{`"type": "basic"`, `"key": "username"`, `"value": "u"`, `"key": "password"`}},
		{"api key defaults to header", request.Request{Protocol: request.ProtocolREST, Method: "GET", Auth: request.Auth{Type: request.AuthAPIKey, KeyName: "X-K", KeyValue: "{{k}}"}},
			[]string{`"type": "apikey"`, `"key": "in"`, `"value": "header"`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := &request.Folder{Name: "C"}
			req := tc.request
			req.Name = "R"
			root.AddRequest(req)
			data, err := Export(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(data), want) {
					t.Errorf("export lacks %s:\n%s", want, data)
				}
			}
		})
	}
}

func TestExportOmitsEmptyOptionalKeys(t *testing.T) {
	root := &request.Folder{Name: "C"}
	root.AddRequest(request.Request{Name: "R", Protocol: request.ProtocolREST, Method: "POST", BodyType: request.BodyText, Body: "x",
		Auth: request.Auth{Type: request.AuthBearer, Token: "{{t}}"}})
	data, _ := Export(root, nil)
	for _, absent := range []string{`"options"`, `"urlencoded"`, `"formdata"`, `"basic"`, `"apikey"`, `"src"`} {
		if strings.Contains(string(data), absent) {
			t.Errorf("export has %s:\n%s", absent, data)
		}
	}
}

func TestImportReadsAnInheritedAuthForEveryRequest(t *testing.T) {
	data := `{"info":{"name":"C","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
	"auth":{"type":"bearer","bearer":[{"key":"token","value":"abc"}]},
	"item":[{"name":"A","request":{"method":"GET","url":"http://x"}},
	{"name":"B","request":{"method":"GET","url":"http://x","auth":{"type":"basic","basic":[{"key":"username","value":"u"}]}}}]}`
	res, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	a, b := find(res.Folder, "A"), find(res.Folder, "B")
	if a.Auth.Type != request.AuthBearer || a.Auth.Token != "abc" || b.Auth.Type != request.AuthBasic || b.Auth.User != "u" {
		t.Fatalf("A %+v B %+v", a.Auth, b.Auth)
	}
}
