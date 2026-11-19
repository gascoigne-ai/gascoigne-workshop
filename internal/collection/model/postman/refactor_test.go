package postman

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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
		request model.Request
		want    []string
	}{
		{"json body", model.Request{Protocol: model.ProtocolREST, Method: "POST", BodyType: model.BodyJSON, Body: "{}"},
			[]string{`"mode": "raw"`, `"language": "json"`}},
		{"text body", model.Request{Protocol: model.ProtocolREST, Method: "POST", BodyType: model.BodyText, Body: "x"},
			[]string{`"mode": "raw"`, `"raw": "x"`}},
		{"form body", model.Request{Protocol: model.ProtocolREST, Method: "POST", BodyType: model.BodyForm, Form: []model.KV{{Name: "a", Value: "1", Enabled: true}}},
			[]string{`"mode": "urlencoded"`, `"key": "a"`}},
		{"multipart body", model.Request{Protocol: model.ProtocolREST, Method: "POST", BodyType: model.BodyMultipart, Form: []model.KV{{Name: "f", Value: "@/tmp/x", Enabled: true}}},
			[]string{`"mode": "formdata"`, `"type": "file"`, `"src": "/tmp/x"`}},
		{"bearer", model.Request{Protocol: model.ProtocolREST, Method: "GET", Auth: model.Auth{Type: model.AuthBearer, Token: "{{t}}"}},
			[]string{`"type": "bearer"`, `"key": "token"`, `"value": "{{t}}"`}},
		{"basic", model.Request{Protocol: model.ProtocolREST, Method: "GET", Auth: model.Auth{Type: model.AuthBasic, User: "u", Password: "plain"}},
			[]string{`"type": "basic"`, `"key": "username"`, `"value": "u"`, `"key": "password"`}},
		{"api key defaults to header", model.Request{Protocol: model.ProtocolREST, Method: "GET", Auth: model.Auth{Type: model.AuthAPIKey, KeyName: "X-K", KeyValue: "{{k}}"}},
			[]string{`"type": "apikey"`, `"key": "in"`, `"value": "header"`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := &model.Folder{Name: "C"}
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
	root := &model.Folder{Name: "C"}
	root.AddRequest(model.Request{Name: "R", Protocol: model.ProtocolREST, Method: "POST", BodyType: model.BodyText, Body: "x",
		Auth: model.Auth{Type: model.AuthBearer, Token: "{{t}}"}})
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
	if a.Auth.Type != model.AuthBearer || a.Auth.Token != "abc" || b.Auth.Type != model.AuthBasic || b.Auth.User != "u" {
		t.Fatalf("A %+v B %+v", a.Auth, b.Auth)
	}
}
