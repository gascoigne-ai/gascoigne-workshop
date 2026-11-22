package request

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestProtoErrorNamesFileLineAndColumn(t *testing.T) {
	err := ProtoError{Path: "/protos/health.proto", Line: 34, Col: 1, Message: "syntax error: unexpected EOF"}
	if got, want := err.Error(), "/protos/health.proto:34:1: syntax error: unexpected EOF"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAnEmptyProtoSourceIsTheServersReflection(t *testing.T) {
	cases := []struct {
		name   string
		source ProtoSource
		want   SourceKind
	}{
		{"nothing", ProtoSource{}, SourceReflection},
		{"import paths alone", ProtoSource{ImportPaths: []string{"/protos"}}, SourceReflection},
		{"a file", ProtoSource{Files: []string{"/protos/health.proto"}}, SourceProtoFiles},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.source.Kind(); got != c.want {
				t.Errorf("Kind() = %q, want %q", got, c.want)
			}
		})
	}
}

// A relative path is saved with "/" on every system, so a collection made
// on Windows opens on Linux and macOS and the other way round; an absolute
// path belongs to one machine and keeps its own separators.
func TestStoredPathIsRelativeWithSlashesInsideTheCollectionAndAbsoluteOutside(t *testing.T) {
	collection := filepath.FromSlash("/work/api")
	cases := []struct {
		name, dir, path, want string
	}{
		{"inside", collection, "/work/api/protos/health.proto", "protos/health.proto"},
		{"deeper inside", collection, "/work/api/protos/grpc/health/v1/health.proto", "protos/grpc/health/v1/health.proto"},
		{"the folder itself", collection, "/work/api", "."},
		{"beside it", collection, "/work/other/health.proto", filepath.FromSlash("/work/other/health.proto")},
		{"a sibling with the same prefix", collection, "/work/api-v2/health.proto", filepath.FromSlash("/work/api-v2/health.proto")},
		{"an unsaved collection", "", "/work/api/health.proto", filepath.FromSlash("/work/api/health.proto")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StoredPath(c.dir, filepath.FromSlash(c.path)); got != c.want {
				t.Errorf("StoredPath(%q, %q) = %q, want %q", c.dir, c.path, got, c.want)
			}
		})
	}
}

// Relative paths are read as saved, with "/", and come back with the
// system's own separators.
func TestLoadedPathTakesRelativeSlashPathsFromTheCollection(t *testing.T) {
	collection := filepath.FromSlash("/work/api")
	cases := []struct {
		name, dir, stored, want string
	}{
		{"relative", collection, "protos/health.proto", "/work/api/protos/health.proto"},
		{"deeper", collection, "protos/grpc/health/v1/health.proto", "/work/api/protos/grpc/health/v1/health.proto"},
		{"absolute", collection, filepath.FromSlash("/work/other/health.proto"), "/work/other/health.proto"},
		{"outside by dots", collection, "../other/health.proto", "/work/other/health.proto"},
		{"an unsaved collection", "", "protos/health.proto", filepath.FromSlash("protos/health.proto")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := LoadedPath(c.dir, c.stored)
			if want := filepath.FromSlash(c.want); got != want {
				t.Errorf("LoadedPath(%q, %q) = %q, want %q", c.dir, c.stored, got, want)
			}
		})
	}
}

func TestAProtoSourceSurvivesStoringAndLoading(t *testing.T) {
	collection := filepath.FromSlash("/work/api")
	source := ProtoSource{
		Files:       []string{filepath.FromSlash("/work/api/protos/health.proto"), filepath.FromSlash("/shared/common.proto")},
		ImportPaths: []string{filepath.FromSlash("/work/api/protos"), filepath.FromSlash("/shared")},
	}
	stored := source.Stored(collection)
	wantStored := ProtoSource{
		Files:       []string{"protos/health.proto", filepath.FromSlash("/shared/common.proto")},
		ImportPaths: []string{"protos", filepath.FromSlash("/shared")},
	}
	if !reflect.DeepEqual(stored, wantStored) {
		t.Errorf("Stored = %+v, want %+v", stored, wantStored)
	}
	if loaded := stored.Loaded(collection); !reflect.DeepEqual(loaded, source) {
		t.Errorf("Loaded = %+v, want %+v", loaded, source)
	}
	if !reflect.DeepEqual(ProtoSource{}.Stored(collection), ProtoSource{}) {
		t.Error("an empty source must store as empty")
	}
}

func TestMoveProtoPathsFollowsTheCollectionFolderEverywhereInTheTree(t *testing.T) {
	from, to := filepath.FromSlash("/work/api"), filepath.FromSlash("/work/office")
	outside := filepath.FromSlash("/shared/common.proto")
	inside := func(dir string) ProtoSource {
		return ProtoSource{Files: []string{filepath.Join(dir, "health.proto"), outside}, ImportPaths: []string{filepath.Join(dir, "protos")}}
	}
	top, nested, plain := &Request{Proto: inside(from)}, &Request{Proto: inside(from)}, &Request{}
	tree := Folder{Requests: []*Request{top, plain}, Folders: []*Folder{{Requests: []*Request{nested}}}}
	tree.MoveProtoPaths(from, to)
	for name, request := range map[string]*Request{"top": top, "nested": nested} {
		if !reflect.DeepEqual(request.Proto, inside(to)) {
			t.Errorf("%s request: %+v, want %+v", name, request.Proto, inside(to))
		}
	}
	if !reflect.DeepEqual(plain.Proto, ProtoSource{}) {
		t.Errorf("a reflection request got proto paths: %+v", plain.Proto)
	}
}

func TestCloneCopiesTheProtoSource(t *testing.T) {
	original := Request{Proto: ProtoSource{Files: []string{"a.proto"}, ImportPaths: []string{"protos"}}}
	clone := original.Clone()
	clone.Proto.Files[0], clone.Proto.ImportPaths[0] = "b.proto", "other"
	if original.Proto.Files[0] != "a.proto" || original.Proto.ImportPaths[0] != "protos" {
		t.Errorf("editing the clone changed the original: %+v", original.Proto)
	}
}
