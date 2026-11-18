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
	filepath.WalkDir(dir, func(name string, entry os.DirEntry, _ error) error {
		if entry.IsDir() || filepath.Base(name) == secretsFile {
			return nil
		}
		b, _ := os.ReadFile(name)
		if strings.Contains(string(b), "s3cret") { // also matches env-s3cret
			t.Errorf("%s contains a secret", name)
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

// A request file whose protocol, body type or auth type is unknown or
// missing stops the collection from loading, as a malformed file does,
// and the error names the file.
func TestLoadRejectsARequestFileWithAnUnknownProtocol(t *testing.T) {
	tests := []struct{ name, data string }{
		{"unknown protocol", `{"name":"x","protocol":"http","method":"GET","url":"u"}`},
		{"no protocol", `{"name":"x","method":"GET","url":"u"}`},
		{"unknown body type", `{"name":"x","protocol":"rest","method":"GET","url":"u","bodyType":"xml"}`},
		{"unknown auth type", `{"name":"x","protocol":"rest","method":"GET","url":"u","auth":{"type":"oauth"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "odd.req.json"), []byte(test.data), 0o644)
			if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "odd.req.json") {
				t.Fatalf("err = %v, want one that names odd.req.json", err)
			}
		})
	}
}

// Files saved while "none" was the stored name of no authentication
// still load, as no authentication.
func TestLoadReadsTheOldNoneAuthTypeAsNoAuthentication(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "old.req.json"), []byte(`{"name":"x","protocol":"rest","method":"GET","url":"u","auth":{"type":"none"}}`), 0o644)
	collection, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := collection.Requests[0].Auth.Type; got != model.AuthNone {
		t.Fatalf("auth type %q, want none", got)
	}
}

func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	filepath.WalkDir(dir, func(name string, entry os.DirEntry, _ error) error {
		if !entry.IsDir() {
			b, _ := os.ReadFile(name)
			m[name] = string(b)
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

// A folder named "Environments" got the directory the environments are
// written to, and saving the environments pruned its subfolders: the
// user's requests were deleted on every save.
func TestAFolderNamedEnvironmentsSurvivesSavingEnvironments(t *testing.T) {
	dir := t.TempDir()
	want := model.Collection{
		Folder: model.Folder{
			Name: "API",
			Folders: []*model.Folder{{
				Name:     "Environments",
				Requests: []*model.Request{{Name: "Top", Protocol: "rest", Method: "GET", URL: "https://x.test/top"}},
				Folders:  []*model.Folder{{Name: "Inner", Requests: []*model.Request{{Name: "Deep", Protocol: "rest", Method: "GET", URL: "https://x.test/deep"}}}},
			}},
		},
		Envs: []model.Environment{{Name: "Dev", Vars: []model.KV{{Name: "base", Value: "http://localhost", Enabled: true}}}},
	}
	for range 2 { // the second save is the one that pruned
		if err := Save(dir, want); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("after saving twice:\n got %+v\nwant %+v", got, want)
	}
}

// .secrets.json holds tokens and secret variable values but was written
// world-readable (0644).
func TestSecretsFileIsOnlyReadableByTheOwner(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, secretsFile))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("%s mode %o, want 600", secretsFile, mode)
	}
}

// A .secrets.json an older version wrote as 0644 is tightened on the next
// save even when its content did not change.
func TestAnOldWorldReadableSecretsFileIsTightenedOnSave(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, secretsFile)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("mode %o after an unchanged save, want 600", mode)
	}
}

// Collection and folder variables marked Secret were written with their
// value into folder.json, which is meant to be committed; only
// environments diverted secrets to .secrets.json.
func TestSecretVariablesOfCollectionsAndFoldersStayOutOfFolderFiles(t *testing.T) {
	dir := t.TempDir()
	want := model.Collection{
		Folder: model.Folder{
			Name: "API",
			Folders: []*model.Folder{{
				Name: "Sub", Vars: []model.KV{{Name: "folderKey", Value: "folder-secret-value", Enabled: true, Secret: true}, {Name: "plain", Value: "visible", Enabled: true}},
				Requests: []*model.Request{{Name: "R", Protocol: "rest", Method: "GET", URL: "u"}},
			}},
		},
		Vars: []model.KV{{Name: "apiKey", Value: "collection-secret-value", Enabled: true, Secret: true}},
	}
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"folder.json", filepath.Join("sub", "folder.json")} {
		text, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(text), "-secret-value") {
			t.Errorf("%s holds a secret value:\n%s", file, text)
		}
	}
	secrets, err := os.ReadFile(filepath.Join(dir, secretsFile))
	if err != nil || !strings.Contains(string(secrets), "collection-secret-value") || !strings.Contains(string(secrets), "folder-secret-value") {
		t.Errorf("%s = %s, %v; want both values", secretsFile, secrets, err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", got, want)
	}
}

// A folder.json an older version wrote still holds the secret value; it
// loads as it was and moves to .secrets.json on the next save.
func TestAnOlderFolderFileWithASecretValueStillLoads(t *testing.T) {
	dir := t.TempDir()
	old := `{"name": "Old", "vars": [{"name": "k", "value": "legacy", "enabled": true, "secret": true}]}`
	if err := os.WriteFile(filepath.Join(dir, "folder.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil || len(got.Vars) != 1 || got.Vars[0].Value != "legacy" {
		t.Fatalf("loaded %+v, %v; want the legacy value kept", got.Vars, err)
	}
	if err := Save(dir, got); err != nil {
		t.Fatal(err)
	}
	if text, _ := os.ReadFile(filepath.Join(dir, "folder.json")); strings.Contains(string(text), "legacy") {
		t.Errorf("the value stayed in folder.json:\n%s", text)
	}
}

// A request with a credential moved to another saved collection takes the
// secret with it: the source's .secrets.json loses the entry, the target's
// has it and its .gitignore lists the file.
func TestMovingACredentialRequestMovesItsSecret(t *testing.T) {
	sourceDir, targetDir := t.TempDir(), t.TempDir()
	source := model.Collection{Folder: model.Folder{Name: "Source"}}
	moved := source.AddRequest(model.Request{Name: "Call", Protocol: "rest", Method: "GET", URL: "http://x.test",
		Auth: model.Auth{Type: model.AuthBearer, Token: "moved-s3cret"}})
	target := model.Collection{Folder: model.Folder{Name: "Target"}}
	if err := Save(sourceDir, source); err != nil {
		t.Fatal(err)
	}
	if err := Save(targetDir, target); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(sourceDir, secretsFile)); !strings.Contains(string(b), "moved-s3cret") {
		t.Fatal("setup: the source has no secret to move")
	}

	source.Delete(moved)
	target.Attach(model.Of(moved), model.Node{})
	if err := Save(targetDir, target); err != nil {
		t.Fatal(err)
	}
	if err := Save(sourceDir, source); err != nil {
		t.Fatal(err)
	}

	if b, err := os.ReadFile(filepath.Join(sourceDir, secretsFile)); err == nil && strings.Contains(string(b), "moved-s3cret") {
		t.Fatalf("the source still holds the secret: %s", b)
	}
	if b, err := os.ReadFile(filepath.Join(targetDir, secretsFile)); err != nil || !strings.Contains(string(b), "moved-s3cret") {
		t.Fatalf("the target lacks the secret: %s %v", b, err)
	}
	if b, _ := os.ReadFile(filepath.Join(targetDir, ".gitignore")); !strings.Contains(string(b), secretsFile) {
		t.Fatalf("the target's .gitignore %q does not list %s", b, secretsFile)
	}
	got, err := Load(targetDir)
	if err != nil || got.Requests[0].Auth.Token != "moved-s3cret" {
		t.Fatalf("target loads %+v %v", got.Requests, err)
	}
}
