package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestProtoPathsAreSavedRelativeInsideTheCollectionAndAbsoluteOutside(t *testing.T) {
	dir, elsewhere := t.TempDir(), t.TempDir()
	source := model.ProtoSource{
		Files:       []string{filepath.Join(dir, "protos", "health.proto"), filepath.Join(elsewhere, "common.proto")},
		ImportPaths: []string{filepath.Join(dir, "protos"), elsewhere},
	}
	collection := model.Collection{Folder: model.Folder{Name: "gRPC", Requests: []*model.Request{
		{Name: "Check", Protocol: "grpc", Method: "grpc.health.v1.Health/Check", URL: "grpc://localhost:8085", Proto: source},
	}}}
	if err := Save(dir, collection); err != nil {
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
	wantFiles := []string{filepath.Join("protos", "health.proto"), filepath.Join(elsewhere, "common.proto")}
	if !reflect.DeepEqual(saved.ProtoFiles, wantFiles) || !reflect.DeepEqual(saved.ImportPaths, []string{"protos", elsewhere}) {
		t.Errorf("saved %+v, want files %v and import paths [protos %s]", saved, wantFiles, elsewhere)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Requests[0].Proto; !reflect.DeepEqual(got, source) {
		t.Errorf("loaded %+v, want %+v", got, source)
	}
}

func TestAReflectionRequestSavesNoProtoKeys(t *testing.T) {
	dir := t.TempDir()
	collection := model.Collection{Folder: model.Folder{Name: "gRPC", Requests: []*model.Request{
		{Name: "Check", Protocol: "grpc", URL: "grpc://localhost:8085"},
	}}}
	if err := Save(dir, collection); err != nil {
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
