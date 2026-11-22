package filedialog

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
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
	t.Setenv("HOME", "/home/me") // the dialog starts in home, like the in-app browser
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
	if !strings.Contains(string(args), "--file-selection") || !strings.Contains(string(args), "--title=Pick it") ||
		!strings.Contains(string(args), "--filename=/home/me/") {
		t.Fatalf("args %q", args)
	}
	if _, err := Save(context.Background(), "Save it", "out.json"); err != nil {
		t.Fatal(err)
	}
	args, _ = os.ReadFile(filepath.Join(os.Getenv("PATH"), "zenity.args"))
	if !strings.Contains(string(args), "--save") || !strings.Contains(string(args), "--filename=/home/me/out.json") {
		t.Fatalf("save args %q", args)
	}
}

// Choose with SaveFile runs the program's save command line, with OpenFile
// its open one.
func TestChooseRunsTheCommandLineOfItsMode(t *testing.T) {
	t.Setenv("HOME", "/home/me")
	tests := map[string]struct {
		mode workspace.FileMode
		want string
	}{
		"save": {workspace.SaveFile, "--save"},
		"open": {workspace.OpenFile, "--file-selection"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fake(t, `echo "$@" > "$0.args"
echo /home/me/file.txt`)
			if _, err := (System{}).Choose(context.Background(), "T", test.mode, "out.json"); err != nil {
				t.Fatal(err)
			}
			args, _ := os.ReadFile(filepath.Join(os.Getenv("PATH"), "zenity.args"))
			if got := strings.Contains(string(args), "--save"); got != (test.mode == workspace.SaveFile) {
				t.Fatalf("mode %v ran %q", test.mode, args)
			}
		})
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
	noDesktop(t)
	if Available() {
		t.Fatal("found a dialog program in an empty PATH")
	}
	if _, err := Open(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "zenity, kdialog, yad, qarma, matedialog") {
		t.Fatalf("err = %v", err)
	}
}

func TestOnAMacTheNativePanelIsRunThroughOsascript(t *testing.T) {
	goos = "darwin"
	t.Cleanup(func() { goos = runtime.GOOS })
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$0.args\"\necho /Users/me/a.json\n"
	if err := os.WriteFile(filepath.Join(dir, "osascript"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := Save(context.Background(), `Say "hi"`, "out.json")
	if err != nil || got != "/Users/me/a.json" {
		t.Fatalf("%q %v", got, err)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "osascript.args"))
	lines := strings.Split(strings.TrimSpace(string(args)), "\n")
	if n := len(lines); n < 2 || lines[n-2] != `Say "hi"` || lines[n-1] != "out.json" {
		t.Fatalf("title and name must be script arguments: %q", args)
	}
	if !strings.Contains(string(args), "choose file name") {
		t.Fatalf("save must use the save panel: %q", args)
	}
}

func TestOnWindowsPowerShellGetsTheTitleAndNameThroughTheEnvironment(t *testing.T) {
	goos = "windows"
	t.Cleanup(func() { goos = runtime.GOOS })
	dir := t.TempDir()
	script := "#!/bin/sh\necho \"$WS_SAVE|$WS_TITLE|$WS_NAME\" > \"$0.env\"\nprintf 'C:\\\\a.json'\n"
	if err := os.WriteFile(filepath.Join(dir, "powershell"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := Save(context.Background(), `Say "hi"`, "out.json"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "powershell.env"))
	if strings.TrimSpace(string(got)) != `1|Say "hi"|out.json` {
		t.Fatalf("env %q", got)
	}
	if _, err := Open(context.Background(), "Pick"); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "powershell.env"))
	if !strings.HasPrefix(string(got), "0|Pick|") {
		t.Fatalf("open env %q", got)
	}
}
