package proto

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/reporter"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// compileProtoFiles reads .proto files into linked descriptors. Relative
// files and import paths are taken from baseDir (the collection's folder).
// Imports are looked up on the import paths in order, then in the folders
// of the given files, then among the standard google/protobuf files.
// Errors in the sources come back as request.ProtoError values, joined, so
// every problem found is shown with its file and line.
func compileProtoFiles(ctx context.Context, baseDir string, files, importPaths []string) (linker.Files, error) {
	files, importPaths = fromBase(baseDir, files), fromBase(baseDir, importPaths)
	names, searchPaths := protoNames(files, importPaths)
	finder := &protoFinder{searchPaths: searchPaths, found: map[string]string{}}
	var problems []error
	collect := func(problem reporter.ErrorWithPos) error {
		problems = append(problems, finder.protoError(problem))
		return nil // keep going: one pass shows every problem
	}
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(finder),
		Reporter: reporter.NewReporter(collect, nil),
	}
	compiled, err := compiler.Compile(ctx, names...)
	if len(problems) > 0 {
		return nil, errors.Join(problems...)
	}
	// A missing import stops the compile without passing the reporter.
	var located reporter.ErrorWithPos
	if errors.As(err, &located) {
		return nil, finder.protoError(located)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", strings.Join(files, ", "), err)
	}
	return compiled, nil
}

// fromBase makes relative paths absolute below baseDir.
func fromBase(baseDir string, paths []string) []string {
	resolved := make([]string, 0, len(paths))
	for _, path := range paths {
		if baseDir != "" && !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		resolved = append(resolved, filepath.Clean(path))
	}
	return resolved
}

// protoNames gives each file the name protoc would: its path below the
// first import path that holds it. Imports between files are written with
// these names, so they must match. A file outside every import path is
// named by its base name, and its folder is searched after the import
// paths, as protoc does when it is run in that folder.
func protoNames(files, importPaths []string) (names, searchPaths []string) {
	searchPaths = append(searchPaths, importPaths...)
	for _, file := range files {
		name, ok := nameBelow(importPaths, file)
		if !ok {
			name = filepath.Base(file)
			if dir := filepath.Dir(file); !containsPath(searchPaths, dir) {
				searchPaths = append(searchPaths, dir)
			}
		}
		names = append(names, name)
	}
	return names, searchPaths
}

// nameBelow is file's slash-separated path below the first of dirs that
// holds it.
func nameBelow(dirs []string, file string) (string, bool) {
	for _, dir := range dirs {
		relative, err := filepath.Rel(dir, file)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		return filepath.ToSlash(relative), true
	}
	return "", false
}

func containsPath(paths []string, wanted string) bool {
	for _, candidate := range paths {
		if candidate == wanted {
			return true
		}
	}
	return false
}

// protoFinder finds .proto sources by name on its search paths and
// remembers where each came from, so an error names the real file.
type protoFinder struct {
	searchPaths []string

	mu    sync.Mutex // the compiler looks files up from several goroutines
	found map[string]string
}

// FindFileByPath implements protocompile.Resolver.
func (f *protoFinder) FindFileByPath(name string) (protocompile.SearchResult, error) {
	if cleaned := path.Clean(name); cleaned == ".." || strings.HasPrefix(cleaned, "../") || path.IsAbs(cleaned) {
		return protocompile.SearchResult{}, fmt.Errorf("%s leaves the import paths: %w", name, fs.ErrNotExist)
	}
	for _, dir := range f.searchPaths {
		path := filepath.Join(dir, filepath.FromSlash(name))
		source, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return protocompile.SearchResult{}, err
		}
		f.mu.Lock()
		f.found[name] = path
		f.mu.Unlock()
		return protocompile.SearchResult{Source: bytes.NewReader(source)}, nil
	}
	return protocompile.SearchResult{}, fmt.Errorf("%s is in none of %s: %w", name, strings.Join(f.searchPaths, ", "), fs.ErrNotExist)
}

// protoError is a located problem of the compiler, with the path of the
// file it is in.
func (f *protoFinder) protoError(problem reporter.ErrorWithPos) request.ProtoError {
	position := problem.GetPosition()
	return request.ProtoError{
		Path:    f.pathOf(position.Filename),
		Line:    position.Line,
		Col:     position.Col,
		Message: problem.Unwrap().Error(),
	}
}

// pathOf is the file a name was read from, or the name when it was never
// read (a missing root file, a standard import).
func (f *protoFinder) pathOf(name string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if path, ok := f.found[name]; ok {
		return path
	}
	return name
}
