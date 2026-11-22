package userdir

import (
	"os"
	"path/filepath"
	"testing"
)

// sandbox points every place a config directory could come from at a
// temp dir, so no test touches real user state.
func sandbox(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("AppData", filepath.Join(root, "config"))
	return filepath.Join(root, "config")
}

func TestPathIsInsideTheProgramsConfigDirectory(t *testing.T) {
	config := sandbox(t)
	tests := []struct {
		name string
		want string
	}{
		{"a file", filepath.Join(config, Name, "state.json")},
		{"the directory itself", filepath.Join(config, Name)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name := ""
			if test.name == "a file" {
				name = "state.json"
			}
			if got := Path(name); got != test.want {
				t.Fatalf("Path(%q) = %q, want %q", name, got, test.want)
			}
		})
	}
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	found, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range found {
		names = append(names, entry.Name())
	}
	return names
}

func TestWriteFileAtomicCreatesAndReplacesTheFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	for _, content := range []string{"first", "second, longer"} {
		if err := WriteFileAtomic(target, []byte(content), 0o640); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(target)
		if err != nil || string(got) != content {
			t.Fatalf("file = %q, %v; want %q", got, err, content)
		}
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want 0640", info.Mode().Perm())
	}
	if names := entries(t, dir); len(names) != 1 || names[0] != "state.json" {
		t.Fatalf("directory holds %v, want only state.json", names)
	}
}

func TestWriteFileAtomicLeavesNoTempFileWhenTheRenameFails(t *testing.T) {
	dir := t.TempDir()
	// A directory cannot be replaced by a file, so the rename fails.
	target := filepath.Join(dir, "taken")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(target, []byte("x"), 0o600); err == nil {
		t.Fatal("want an error")
	}
	if names := entries(t, dir); len(names) != 1 || names[0] != "taken" {
		t.Fatalf("directory holds %v, want only taken", names)
	}
}

func TestWriteFileAtomicFailsWithoutTheDirectory(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing", "file")
	if err := WriteFileAtomic(target, []byte("x"), 0o600); err == nil {
		t.Fatal("want an error")
	}
}
