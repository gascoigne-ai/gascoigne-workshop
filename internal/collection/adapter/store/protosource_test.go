package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func TestProtoPathsAreSavedRelativeInsideTheCollectionAndAbsoluteOutside(t *testing.T) {
	dir, elsewhere := t.TempDir(), t.TempDir()
	source := request.ProtoSource{
		Files:       []string{filepath.Join(dir, "protos", "health.proto"), filepath.Join(elsewhere, "common.proto")},
		ImportPaths: []string{filepath.Join(dir, "protos"), elsewhere},
	}
	collection := request.Collection{Folder: request.Folder{Name: "gRPC", Requests: []*request.Request{
		{Name: "Check", Protocol: "grpc", Method: "grpc.health.v1.Health/Check", URL: "grpc://localhost:8085", Proto: source},
	}}}
	if err := disk.Save(dir, collection); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "check"+reqSuffix))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		ProtoFiles  []string `json:"protoFiles"`
		ImportPaths []string `json:"importPaths"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	wantFiles := []string{"protos/health.proto", filepath.Join(elsewhere, "common.proto")}
	if !reflect.DeepEqual(saved.ProtoFiles, wantFiles) || !reflect.DeepEqual(saved.ImportPaths, []string{"protos", elsewhere}) {
		t.Errorf("saved %+v, want files %v and import paths [protos %s]", saved, wantFiles, elsewhere)
	}

	loaded, err := disk.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Requests[0].Proto; !reflect.DeepEqual(got, source) {
		t.Errorf("loaded %+v, want %+v", got, source)
	}
}

// A collection opened by a relative folder (a -workspace flag) saved the
// proto paths inside it as absolute ones, since a relative folder and an
// absolute path have no relative path between them.
func TestProtoPathsAreRelativeWhenTheCollectionFolderIsGivenRelative(t *testing.T) {
	parent := t.TempDir()
	t.Chdir(parent)
	inside := filepath.Join(parent, "api", "health.proto")
	collection := request.Collection{Folder: request.Folder{Name: "gRPC", Requests: []*request.Request{
		{Name: "Check", Protocol: "grpc", Proto: request.ProtoSource{Files: []string{inside}}},
	}}}
	if err := disk.Save("api", collection); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(parent, "api", "check"+reqSuffix))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		ProtoFiles []string `json:"protoFiles"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.ProtoFiles, []string{"health.proto"}) {
		t.Errorf("saved %v, want [health.proto]", saved.ProtoFiles)
	}
	loaded, err := disk.Load("api")
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Requests[0].Proto.Files; !reflect.DeepEqual(got, []string{inside}) {
		t.Errorf("loaded %v, want [%s]", got, inside)
	}
}

func TestAReflectionRequestSavesNoProtoKeys(t *testing.T) {
	dir := t.TempDir()
	collection := request.Collection{Folder: request.Folder{Name: "gRPC", Requests: []*request.Request{
		{Name: "Check", Protocol: "grpc", URL: "grpc://localhost:8085"},
	}}}
	if err := disk.Save(dir, collection); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "check"+reqSuffix))
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"protoFiles", "importPaths"} {
		if _, ok := keys[key]; ok {
			t.Errorf("%s is saved for a request that uses reflection", key)
		}
	}
}
