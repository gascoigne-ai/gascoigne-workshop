package app

import (
	"bytes"
	"context"
	"errors"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/importers/postman"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fakeFolders is a file system in memory. Writes arrive from background
// goroutines, so every field is behind the lock.
type fakeFolders struct {
	mu       sync.Mutex
	home     string
	files    map[string][]byte
	writeErr error
	checked  []string      // the paths IsFile was asked about, in order
	release  chan struct{} // when set, Write waits for it to close
}

func newFakeFolders() *fakeFolders {
	return &fakeFolders{home: "/home/user", files: map[string][]byte{}}
}

func (f *fakeFolders) List(string) ([]workspace.FolderEntry, error) { return nil, nil }
func (f *fakeFolders) Home() (string, error)                        { return f.home, nil }

func (f *fakeFolders) Write(_ context.Context, path string, data []byte) error {
	if f.release != nil {
		<-f.release
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return f.writeErr
	}
	f.files[path] = bytes.Clone(data)
	return nil
}

func (f *fakeFolders) IsFile(_ context.Context, path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checked = append(f.checked, path)
	_, ok := f.files[path]
	return ok
}

func (f *fakeFolders) file(path string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.files[path]
	return data, ok
}

func (f *fakeFolders) checks(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, checked := range f.checked {
		if checked == path {
			n++
		}
	}
	return n
}

func TestASavedBodyIsWrittenThroughTheFileSystemWithTildeAsHome(t *testing.T) {
	a := testApp(t)
	files := newFakeFolders()
	a.folders = files
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte(`{"a":1}`)}
	a.writeBody(tab, "~/out.json")
	until(t, a, "the saved toast", func() bool { return toastWith(a, "out.json", false) })
	if data, ok := files.file("/home/user/out.json"); !ok || string(data) != `{"a":1}` {
		t.Fatalf("written %q %v, want the body in the home folder", data, ok)
	}
	if !toastWith(a, "Saved 7B to out.json", false) {
		t.Fatalf("toasts %+v", a.toasts)
	}
}

func TestASavedBodyThatCannotBeWrittenIsReported(t *testing.T) {
	a := testApp(t)
	files := newFakeFolders()
	files.writeErr = errors.New("disk full")
	a.folders = files
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("x")}
	a.writeBody(tab, "/out/body.bin")
	until(t, a, "the failure toast", func() bool { return toastWith(a, "Saving failed: disk full", true) })
}

// A write to a slow disk must not hold the window: the frame goes on while
// the file is written, and the toast comes when it is done.
func TestSavingABodyDoesNotWaitForTheDisk(t *testing.T) {
	a := testApp(t)
	files := newFakeFolders()
	files.release = make(chan struct{})
	a.folders = files
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("x")}
	a.writeBody(tab, "/out/body.bin") // would block here if written on the UI goroutine
	if toastWith(a, "Saved", false) {
		t.Fatal("saved before the disk took the file")
	}
	close(files.release)
	until(t, a, "the saved toast", func() bool { return toastWith(a, "Saved 1B to body.bin", false) })
}

func TestAPostmanExportIsWrittenThroughTheFileSystem(t *testing.T) {
	a := testApp(t)
	files := newFakeFolders()
	a.folders = files
	a.exportPostman(&a.col().Folder)
	typePath(a, "/out/col.json")
	a.browse(filebrowser.Entered{})
	until(t, a, "the export toast", func() bool { return toastWith(a, "Exported to col.json", false) })
	data, _ := files.file("/out/col.json")
	if res, err := postman.Parse(data); err != nil || res.Folder.Name != a.col().Name {
		t.Fatalf("%v %+v", err, res.Folder)
	}
	if !toastWith(a, "left out", false) { // the sample has WebSocket and MCP requests
		t.Fatalf("toasts %+v", a.toasts)
	}
}

func TestAPostmanExportThatCannotBeWrittenIsReported(t *testing.T) {
	a := testApp(t)
	files := newFakeFolders()
	files.writeErr = errors.New("disk full")
	a.folders = files
	a.exportPostman(&a.col().Folder)
	typePath(a, "/out/col.json")
	a.browse(filebrowser.Entered{})
	until(t, a, "the failure toast", func() bool { return toastWith(a, "Export failed: disk full", true) })
}

// The editor of a file body is built every frame; the file system is asked
// about a path once, off the frame, not once per frame.
func TestTheFileBodyAsksAboutEachPathOnce(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"a file that is there", "/data/found.bin", `text="found"`},
		{"a file that is not", "/data/missing.bin", `text="not found"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := openFileBody(t)
			files := newFakeFolders()
			files.files["/data/found.bin"] = []byte("data")
			a.folders = files
			tab := a.cur()
			tab.file = widget.NewTextField(test.path)
			until(t, a, test.want, func() bool { return strings.Contains(view.Dump(a.fileBody(tab)), test.want) })
			for range 5 {
				view.Dump(a.fileBody(tab))
				a.step(core.Input{})
			}
			if n := files.checks(test.path); n != 1 {
				t.Fatalf("asked %d times about %s, want once", n, test.path)
			}
		})
	}
}

// The composition root reaches files only through ports, so a slow or
// broken disk never holds the window and every access has a fake in tests.
func TestTheAppDoesNoFileAccessOfItsOwn(t *testing.T) {
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), source, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			if path, _ := strconv.Unquote(imported.Path.Value); path == "os" {
				t.Errorf("%s imports os; reach files through a port (workspace.Folders)", source)
			}
		}
	}
}
