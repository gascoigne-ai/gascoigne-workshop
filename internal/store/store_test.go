package store

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func sample() model.Collection {
	return model.Collection{
		Folder: model.Folder{
			Name: "Café API ☕",
			Requests: []*model.Request{
				{Name: "Ping", Protocol: "rest", Method: "GET", URL: "https://x.test/ping",
					Params:  []model.KV{{Name: "q", Value: "a b", Enabled: true}, {Name: "off", Value: "1", Description: "disabled"}},
					Headers: []model.KV{{Name: "Accept", Value: "application/json", Enabled: true}}},
			},
			Folders: []*model.Folder{
				{Name: "Users", Requests: []*model.Request{
					{Name: "Create", Protocol: "rest", Method: "POST", URL: "{{base}}/users", BodyType: model.BodyJSON, Body: "{\n  \"a\": 1\n}",
						Auth: model.Auth{Type: model.AuthBearer, Token: "s3cret"}, TimeoutSec: 5, NoRedirects: true, InsecureTLS: true, Tests: "status == 201\njson id exists"},
					{Name: "Create", Protocol: "rest", Method: "PUT", URL: "{{base}}/users/1"}, // same name twice
				}},
				{Name: "Empty", Vars: []model.KV{{Name: "folder-var", Value: "fv", Enabled: true}}},
				{Name: "a/b: weird?*", Requests: []*model.Request{{Name: "Echo", Protocol: "websocket", URL: "wss://e.test"}}},
			},
		},
		Vars: []model.KV{{Name: "base", Value: "https://x.test", Enabled: true}},
		Envs: []model.Environment{
			{Name: "Dev", Vars: []model.KV{{Name: "base", Value: "http://localhost", Enabled: true}, {Name: "token", Value: "env-s3cret", Enabled: true, Secret: true}}},
			{Name: "Prod"},
		},
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	want := sample()
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip differs:\n got %+v\nwant %+v", got, want)
	}
}

func TestSecretsStayOutOfRequestFiles(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, _ error) error {
		if d.IsDir() || filepath.Base(p) == secretsFile {
			return nil
		}
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), "s3cret") { // also matches env-s3cret
			t.Errorf("%s contains a secret", p)
		}
		return nil
	})
	ig, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(ig), secretsFile) {
		t.Fatalf(".gitignore = %q", ig)
	}
}

func TestSaveIsStableAndRemovesDeletedRequests(t *testing.T) {
	dir := t.TempDir()
	c := sample()
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	first := snapshot(t, dir)
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, snapshot(t, dir)) {
		t.Fatal("saving twice changed files")
	}
	c.Folders[0].Requests = c.Folders[0].Requests[:1]
	c.Folders = c.Folders[:1] // drops Empty and the weird folder
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, c) {
		t.Fatalf("after deletion:\n got %+v\nwant %+v", got, c)
	}
}

func TestSaveKeepsFilesItDoesNotOwn(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "README.md")
	os.WriteFile(note, []byte("mine"), 0o644)
	if err := Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(note); string(b) != "mine" {
		t.Fatal("foreign file touched")
	}
}

func TestLoadOfAnEmptyDirectoryIsAnEmptyCollection(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil || len(c.Requests) != 0 || len(c.Folders) != 0 {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestLoadKeepsRequestsWhoseOrderEntryIsMissing(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "added-by-hand.req.json"), []byte(`{"name":"Hand","protocol":"rest","method":"GET","url":"u"}`), 0o644)
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	last := c.Requests[len(c.Requests)-1]
	if last.Name != "Hand" {
		t.Fatalf("hand-made request missing: %+v", c.Requests)
	}
}

func TestLoadReportsBrokenFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.req.json"), []byte("{nope"), 0o644)
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "bad.req.json") {
		t.Fatalf("err = %v", err)
	}
}

func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, _ error) error {
		if !d.IsDir() {
			b, _ := os.ReadFile(p)
			m[p] = string(b)
		}
		return nil
	})
	return m
}

func TestDeletedEnvironmentsLeaveNoFiles(t *testing.T) {
	dir := t.TempDir()
	c := sample()
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	c.Envs = c.Envs[1:] // drop Dev
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil || !reflect.DeepEqual(got.Envs, c.Envs) {
		t.Fatalf("%+v %v", got.Envs, err)
	}
	c.Envs = nil
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, envDir))
	if len(entries) != 0 {
		t.Fatalf("%d environment files left", len(entries))
	}
	if b, _ := os.ReadFile(filepath.Join(dir, secretsFile)); strings.Contains(string(b), "env-s3cret") {
		t.Fatal("secret of a deleted environment kept")
	}
}
