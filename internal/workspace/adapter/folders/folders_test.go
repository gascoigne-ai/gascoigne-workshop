package folders

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
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
	slices.SortFunc(got, func(left, right port.FolderEntry) int {
		if left.Name < right.Name {
			return -1
		}
		return 1
	})
	want := []port.FolderEntry{{Name: "a.json"}, {Name: "broken"}, {Name: "link", Dir: true}, {Name: "sub", Dir: true}}
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
