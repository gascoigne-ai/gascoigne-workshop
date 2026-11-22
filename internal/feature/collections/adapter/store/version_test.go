package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/fileversion"
)

// The fixtures are a collection the program saved: testdata/v0 before files
// carried a version (the example sample with two environments), testdata/v1
// the same collection saved again by this build. Their .gitignore and the
// copied .proto file are left out.

// copyTree copies the fixture src into a fresh folder, so a test can save
// over it.
func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestLoadReadsCollectionFixtures(t *testing.T) {
	for _, version := range []string{"v0", "v1"} {
		t.Run(version, func(t *testing.T) { checkFixtureCollection(t, filepath.Join("testdata", version)) })
	}
}

// checkFixtureCollection loads the fixture collection in dir and checks it
// is the sample that was saved.
func checkFixtureCollection(t *testing.T, dir string) {
	t.Helper()
	got, err := disk.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Example Server" {
		t.Errorf("name = %q", got.Name)
	}
	var names []string
	for _, call := range got.Requests {
		names = append(names, call.Name)
	}
	wantNames := []string{"Index", "Echo", "Status 418", "Delay 500ms", "Auth", "Auth without credentials", "WebSocket Echo", "SSE Ticks", "MCP Tools", "gRPC Health", "gRPC Health of example", "gRPC Health from .proto"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("requests = %q\nwant %q", names, wantNames)
	}
	if auth := got.Requests[4].Auth; auth.Type != request.AuthBearer || auth.Token != "example-token" {
		t.Errorf("Auth = %+v, want the bearer token from .secrets.json", auth)
	}
	root, _ := filepath.Abs(dir)
	if files := got.Requests[11].Proto.Files; !reflect.DeepEqual(files, []string{filepath.Join(root, "protos", "health.proto")}) {
		t.Errorf("proto files = %q, want the collection-relative path made absolute", files)
	}
	wantEnvs := []request.Environment{
		{Name: "Local", Vars: []request.KV{{Name: "host", Value: "127.0.0.1", Enabled: true}, {Name: "apiKey", Value: "example-token", Enabled: true, Secret: true}}},
		{Name: "Staging"},
	}
	if !reflect.DeepEqual(got.Envs, wantEnvs) {
		t.Errorf("environments = %+v\nwant %+v", got.Envs, wantEnvs)
	}
}

// savedFiles are the JSON files of a saved collection, by path below dir.
func savedFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		data, err := os.ReadFile(path)
		rel, _ := filepath.Rel(dir, path)
		files[filepath.ToSlash(rel)] = data
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestSaveWritesVersionInEveryFile(t *testing.T) {
	loaded, err := disk.Load(filepath.Join("testdata", "v0"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := disk.Save(dir, loaded); err != nil {
		t.Fatal(err)
	}
	files := savedFiles(t, dir)
	for _, name := range []string{folderFile, "echo.req.json", "environments/01-local.env.json", secretsFile} {
		if _, ok := files[name]; !ok {
			t.Errorf("%s was not written", name)
		}
	}
	for name, data := range files {
		if want := "{\n  \"version\": 1,\n"; !bytes.HasPrefix(data, []byte(want)) {
			t.Errorf("%s starts %q, want %q", name, data[:min(len(data), 30)], want)
		}
	}
}

func TestSaveWrapsSecretsWithVersion(t *testing.T) {
	loaded, err := disk.Load(filepath.Join("testdata", "v0"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := disk.Save(dir, loaded); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, secretsFile))
	if err != nil {
		t.Fatal(err)
	}
	var wrapped struct {
		Version int                       `json:"version"`
		Files   map[string]map[string]any `json:"files"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		t.Fatal(err)
	}
	if wrapped.Version != 1 || wrapped.Files["auth.req.json"]["token"] != "example-token" {
		t.Fatalf("%s = %s, want version 1 and the secrets under files", secretsFile, data)
	}
	again, err := disk.Load(dir)
	if err != nil || again.Requests[4].Auth.Token != "example-token" {
		t.Fatalf("reloaded token = %q, %v", again.Requests[4].Auth.Token, err)
	}
}

// withVersion is a v1 file with its version replaced by raw, the way a file
// of another build would carry it: the bad versions are not fixture files.
func withVersion(data []byte, raw string) []byte {
	return bytes.Replace(data, []byte(`"version": 1`), []byte(`"version": `+raw), 1)
}

func TestSaveLoadRoundTripsV1Fixture(t *testing.T) {
	fixture := filepath.Join("testdata", "v1")
	dir := copyTree(t, fixture) // saved in place: proto paths stay relative to the folder
	loaded, err := disk.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := disk.Save(dir, loaded); err != nil {
		t.Fatal(err)
	}
	want, got := savedFiles(t, fixture), savedFiles(t, dir)
	if len(got) != len(want) {
		t.Fatalf("saved %d files, the fixture has %d", len(got), len(want))
	}
	for name, data := range want {
		if !bytes.Equal(got[name], data) {
			t.Errorf("%s saved as\n%s\nwant\n%s", name, got[name], data)
		}
	}
}

func TestLoadRejectsBadVersionNamingFile(t *testing.T) {
	files := []string{folderFile, "echo.req.json", "environments/01-local.env.json", secretsFile}
	versions := []struct {
		raw       string
		wantNewer bool
	}{
		{"2", true},
		{"-1", false},
		{`"1"`, false},
	}
	for _, name := range files {
		for _, version := range versions {
			t.Run(name+" "+version.raw, func(t *testing.T) {
				dir := copyTree(t, filepath.Join("testdata", "v1"))
				path := filepath.Join(dir, filepath.FromSlash(name))
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, withVersion(data, version.raw), 0o600); err != nil {
					t.Fatal(err)
				}
				_, err = disk.Load(dir)
				if err == nil {
					t.Fatal("Load gave no error")
				}
				if !strings.Contains(err.Error(), filepath.Base(name)) {
					t.Errorf("error %q does not name %s", err, name)
				}
				var newer *fileversion.NewerError
				if errors.As(err, &newer) != version.wantNewer {
					t.Errorf("error %q: is NewerError = %v, want %v", err, !version.wantNewer, version.wantNewer)
				}
			})
		}
	}
}

func TestRenameWritesTheCurrentVersion(t *testing.T) {
	dir := copyTree(t, filepath.Join("testdata", "v0"))
	moved, err := disk.Rename(dir, "Renamed")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(moved, folderFile))
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"version\": 1,\n  \"name\": \"Renamed\""; !bytes.HasPrefix(data, []byte(want)) {
		t.Fatalf("%s = %s, want it to start %q", folderFile, data, want)
	}
}
