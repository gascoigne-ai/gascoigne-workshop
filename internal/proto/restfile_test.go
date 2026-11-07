package proto

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestFileBodyIsSentAsItIsWithItsTypeFromTheExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`{"a":1}`), 0o644)
	var gotBody, gotType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody, gotType = string(b), r.Header.Get("Content-Type")
	}))
	defer srv.Close()
	resp, _ := REST{}.Send(context.Background(), model.Request{Method: "POST", URL: srv.URL, BodyType: model.BodyFile, Body: path})
	if resp.Err != "" || gotBody != `{"a":1}` || !strings.HasPrefix(gotType, "application/json") {
		t.Fatalf("%+v body %q type %q", resp, gotBody, gotType)
	}
	for body, want := range map[string]string{"": "choose the file", filepath.Join(dir, "missing.bin"): "body file"} {
		_, err := REST{}.Send(context.Background(), model.Request{Method: "POST", URL: srv.URL, BodyType: model.BodyFile, Body: body})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("body %q: err = %v, want %q", body, err, want)
		}
	}
	if _, _, err := encodeBody(model.Request{BodyType: model.BodyFile}); err == nil {
		t.Fatal("an empty path must be an error")
	}
}

func TestMultipartFieldWithAtSignSendsAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	os.WriteFile(path, []byte("hello file"), 0o644)
	req := model.Request{BodyType: model.BodyMultipart, Form: []model.KV{
		{Name: "title", Value: "plain", Enabled: true},
		{Name: "upload", Value: "@" + path, Enabled: true},
		{Name: "literal", Value: "@@at", Enabled: true},
	}}
	body, ct, err := encodeBody(req)
	if err != nil {
		t.Fatal(err)
	}
	_, params, _ := mime.ParseMediaType(ct)
	r := multipart.NewReader(strings.NewReader(string(body)), params["boundary"])
	got := map[string]string{}
	var fileName, fileType string
	for {
		p, err := r.NextPart()
		if err != nil {
			break
		}
		b, _ := io.ReadAll(p)
		got[p.FormName()] = string(b)
		if p.FormName() == "upload" {
			fileName, fileType = p.FileName(), p.Header.Get("Content-Type")
		}
	}
	if got["title"] != "plain" || got["upload"] != "hello file" || got["literal"] != "@at" || fileName != "note.txt" || !strings.HasPrefix(fileType, "text/plain") {
		t.Fatalf("%v name %q type %q", got, fileName, fileType)
	}
	req.Form[1].Value = "@" + filepath.Join(t.TempDir(), "nope")
	if _, _, err := encodeBody(req); err == nil || !strings.Contains(err.Error(), "upload") {
		t.Fatalf("a missing file must be an error naming the field: %v", err)
	}
}
