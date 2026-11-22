package folders

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
)

func TestListTellsFoldersFromFilesAndFollowsLinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A link to a folder is not a folder to os.DirEntry, but the browser
	// must open it like one.
	if err := os.Symlink(filepath.Join(dir, "sub"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	// A link to nothing is listed as a file rather than dropped or failing the list.
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "broken")); err != nil {
		t.Fatal(err)
	}
	got, err := Disk{}.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(got, func(left, right workspace.FolderEntry) int {
		if left.Name < right.Name {
			return -1
		}
		return 1
	})
	want := []workspace.FolderEntry{{Name: "a.json"}, {Name: "broken"}, {Name: "link", Dir: true}, {Name: "sub", Dir: true}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestListOfAMissingFolderIsAnError(t *testing.T) {
	if _, err := (Disk{}).List(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("no error for a missing folder")
	}
}

func TestHomeIsTheUsers(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if home, err := (Disk{}).Home(); err != nil || home != dir {
		t.Fatalf("home %q %v", home, err)
	}
}

func TestWriteReplacesTheFileWithTheData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	if err := os.WriteFile(path, []byte("an older and longer content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (Disk{}).Write(context.Background(), path, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != `{"a":1}` {
		t.Fatalf("read %q %v", data, err)
	}
}

func TestWriteIntoAMissingFolderIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no", "such", "out.json")
	if err := (Disk{}).Write(context.Background(), path, []byte("x")); err == nil {
		t.Fatal("no error for a missing folder")
	}
}

func TestWriteThatWasCancelledWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (Disk{}).Write(ctx, path, []byte("x")); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want cancelled", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the file was written: %v", err)
	}
}

func TestIsFileTellsFilesFromFoldersAndMissingPaths(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"a file", file, true},
		{"a folder", dir, false},
		{"a missing path", filepath.Join(dir, "gone"), false},
		{"an empty path", "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := (Disk{}).IsFile(context.Background(), test.path); got != test.want {
				t.Fatalf("IsFile(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}
