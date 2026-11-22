package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

func TestCreateMakesAnEmptyCollectionInAUniqueFolder(t *testing.T) {
	parent := t.TempDir()
	first, err := disk.Create(parent, "Café API")
	if err != nil {
		t.Fatal(err)
	}
	second, err := disk.Create(parent, "Café API")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "café-api" || filepath.Base(second) != "café-api-2" {
		t.Fatalf("folders %q and %q", first, second)
	}
	got, err := disk.Load(second)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Café API" || len(got.Requests) != 0 {
		t.Fatalf("loaded %+v", got)
	}
}

func TestRenameChangesTheNameAndTheFolderAndKeepsTheContent(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "old")
	if err := disk.Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(parent, "taken"), 0o755); err != nil {
		t.Fatal(err)
	}
	moved, err := disk.Rename(dir, "Taken")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(moved) != "taken-2" {
		t.Fatalf("moved to %q, want taken-2 (taken is in use)", moved)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the old folder is still there: %v", err)
	}
	got, err := disk.Load(moved)
	if err != nil {
		t.Fatal(err)
	}
	want := sample()
	want.Name = "Taken"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("content changed:\n got %+v\nwant %+v", got, want)
	}
}

func TestRenameToTheSameSlugStaysInPlace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "echo")
	if err := disk.Save(dir, request.Collection{Folder: request.Folder{Name: "Echo"}}); err != nil {
		t.Fatal(err)
	}
	moved, err := disk.Rename(dir, "ECHO!")
	if err != nil || moved != dir {
		t.Fatalf("moved to %q, %v; want %q", moved, err, dir)
	}
	if got, _ := disk.Load(dir); got.Name != "ECHO!" {
		t.Fatalf("name %q", got.Name)
	}
}

func TestRenameAndDeleteRefuseAFolderThatIsNotACollection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := disk.Rename(dir, "X"); err == nil {
		t.Fatal("Rename of a plain folder succeeded")
	}
	if err := disk.Delete(dir); err == nil {
		t.Fatal("Delete of a plain folder succeeded")
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatalf("the user's file is gone: %v", err)
	}
}

func TestDeleteRemovesTheCollectionAndOnlyItsFiles(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "c")
	if err := disk.Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := disk.Delete(dir); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "README.md" {
		t.Fatalf("left in the folder: %v", entries)
	}
	if err := os.Remove(filepath.Join(dir, "README.md")); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(parent, "d")
	if err := disk.Save(other, sample()); err != nil {
		t.Fatal(err)
	}
	if err := disk.Delete(other); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(other); !os.IsNotExist(err) {
		t.Fatalf("an emptied collection folder stays: %v", err)
	}
}

func TestListFindsCollectionFoldersByName(t *testing.T) {
	parent := t.TempDir()
	for _, name := range []string{"b", "a"} {
		if err := disk.Save(filepath.Join(parent, name), request.Collection{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(parent, "plain"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := disk.List(parent)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(parent, "a"), filepath.Join(parent, "b")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if none, err := disk.List(filepath.Join(parent, "missing")); err != nil || none != nil {
		t.Fatalf("a missing parent lists %v, %v", none, err)
	}
}

func TestDeleteKeepsAGitignoreWithOtherLines(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "c")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := disk.Save(dir, sample()); err != nil {
		t.Fatal(err)
	}
	if err := disk.Delete(dir); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, ".gitignore")); string(b) != "build/\n"+secretsFile+"\n" {
		t.Fatalf(".gitignore is %q", b)
	}
}
