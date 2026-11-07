package filedialog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fake puts a script called zenity first on the PATH.
func fake(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zenity"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestOpenAndSaveReturnWhatTheDialogPrints(t *testing.T) {
	fake(t, `echo "$@" > "$0.args"
echo /home/me/file.txt`)
	if !Available() {
		t.Fatal("zenity on the PATH must be found")
	}
	got, err := Open(context.Background(), "Pick it")
	if err != nil || got != "/home/me/file.txt" {
		t.Fatalf("%q %v", got, err)
	}
	args, _ := os.ReadFile(filepath.Join(os.Getenv("PATH"), "zenity.args"))
	if !strings.Contains(string(args), "--file-selection") || !strings.Contains(string(args), "--title=Pick it") {
		t.Fatalf("args %q", args)
	}
	if _, err := Save(context.Background(), "Save it", "out.json"); err != nil {
		t.Fatal(err)
	}
	args, _ = os.ReadFile(filepath.Join(os.Getenv("PATH"), "zenity.args"))
	if !strings.Contains(string(args), "--save") || !strings.Contains(string(args), "--filename=out.json") {
		t.Fatalf("save args %q", args)
	}
}

func TestClosingTheDialogIsACancelNotAnError(t *testing.T) {
	fake(t, "exit 1")
	if _, err := Open(context.Background(), "x"); err != ErrCancelled {
		t.Fatalf("err = %v", err)
	}
	fake(t, "exit 0") // prints nothing
	if _, err := Open(context.Background(), "x"); err != ErrCancelled {
		t.Fatalf("an empty answer is a cancel: %v", err)
	}
	fake(t, "exit 3")
	if _, err := Open(context.Background(), "x"); err == nil || err == ErrCancelled {
		t.Fatalf("a crash is an error: %v", err)
	}
}

func TestWithoutADialogProgramNothingIsAvailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if Available() {
		t.Fatal("found a dialog program in an empty PATH")
	}
	if _, err := Open(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "zenity or kdialog") {
		t.Fatalf("err = %v", err)
	}
}
