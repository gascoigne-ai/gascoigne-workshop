package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ProtoSource is where a gRPC request learns the server's services and
// messages: .proto files, with the folders their imports are searched
// in, in order. Without files it is the server's reflection. In memory
// the paths are absolute; in a saved collection they are as StoredPath
// gives them.
type ProtoSource struct {
	Files       []string
	ImportPaths []string
}

// SourceKind says where a ProtoSource's services come from. It is derived
// from the files and never stored.
type SourceKind string

// Kinds of ProtoSource, the keys of the gRPC adapter's table of sources.
const (
	SourceReflection SourceKind = "reflection"
	SourceProtoFiles SourceKind = "proto files"
)

// Kind says where the services come from. Import paths alone change
// nothing: they only serve the files.
func (s ProtoSource) Kind() SourceKind {
	if len(s.Files) == 0 {
		return SourceReflection
	}
	return SourceProtoFiles
}

// Stored is the source as saved in a collection kept in dir.
func (s ProtoSource) Stored(dir string) ProtoSource {
	return ProtoSource{Files: mapPaths(s.Files, dir, StoredPath), ImportPaths: mapPaths(s.ImportPaths, dir, StoredPath)}
}

// Loaded is a source read from a collection kept in dir, with absolute
// paths.
func (s ProtoSource) Loaded(dir string) ProtoSource {
	return ProtoSource{Files: mapPaths(s.Files, dir, LoadedPath), ImportPaths: mapPaths(s.ImportPaths, dir, LoadedPath)}
}

// MoveProtoPaths keeps the proto paths of every request below f that
// were inside the folder from inside the folder to, after the collection
// was moved there; paths outside it stay as they are.
func (f *Folder) MoveProtoPaths(from, to string) {
	for _, request := range f.Requests {
		request.Proto = request.Proto.Stored(from).Loaded(to)
	}
	for _, sub := range f.Folders {
		sub.MoveProtoPaths(from, to)
	}
}

func (s ProtoSource) clone() ProtoSource {
	return ProtoSource{Files: cloneStrings(s.Files), ImportPaths: cloneStrings(s.ImportPaths)}
}

func mapPaths(paths []string, dir string, convert func(dir, path string) string) []string {
	if len(paths) == 0 {
		return nil
	}
	converted := make([]string, len(paths))
	for i, path := range paths {
		converted[i] = convert(dir, path)
	}
	return converted
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

// StoredPath is how path is saved in a collection kept in dir: relative
// to dir when it is inside, so the folder can be moved or shared with
// its files, and absolute when it is outside. A relative path is written
// with "/" so the collection reads the same on every system; an absolute
// one names a place on this machine only and keeps its separators. An
// unsaved collection (dir "") keeps the path as it is.
func StoredPath(dir, path string) string {
	if dir == "" || !filepath.IsAbs(path) {
		return path
	}
	relative, err := filepath.Rel(dir, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(relative)
}

// LoadedPath is the path a stored one stands for in a collection kept in
// dir: a relative path, saved with "/", is taken from dir.
func LoadedPath(dir, stored string) string {
	stored = filepath.FromSlash(stored)
	if dir == "" || filepath.IsAbs(stored) {
		return stored
	}
	return filepath.Join(dir, stored)
}

// ProtoError is a problem in a .proto file, at the line and column where
// it was found (both from 1), so the user can go to it.
type ProtoError struct {
	Path      string
	Line, Col int
	Message   string
}

func (e ProtoError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Col, e.Message)
}
