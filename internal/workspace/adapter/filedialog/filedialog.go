// Package filedialog asks the desktop for a file name by running the
// dialog program the system has: the native panel through osascript on
// macOS, a PowerShell dialog on Windows, and elsewhere the first of zenity,
// kdialog, yad, qarma and matedialog that is installed, in the order the
// desktop prefers (detect.go). Without one, Available is false and callers
// fall back to the in-app file browser.
package filedialog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
)

// ErrCancelled is returned when the user closed the dialog.
var ErrCancelled = port.ErrCancelled

// tool describes how to run one dialog program.
type tool struct {
	name string
	// env adds variables to the program's environment; the Windows script
	// reads the title and the name from them, never from its text.
	env  func(title, name string, save bool) []string
	open func(title string) []string
	save func(title, name string) []string
	// cancels are the exit codes that mean the user closed the dialog
	// rather than that the program failed.
	cancels []int
}

// goos is the operating system; tests change it.
var goos = runtime.GOOS

// osascript runs AppleScript, which is on every Mac, and shows the native
// open and save panels. The title and the name travel as arguments of the
// script, never inside its text. Cancelling makes it exit 1.
var osascript = tool{
	name: "osascript",
	open: func(title string) []string {
		return []string{
			"-e", "on run argv",
			"-e", "POSIX path of (choose file with prompt (item 1 of argv))",
			"-e", "end run", title}
	},
	save: func(title, name string) []string {
		return []string{
			"-e", "on run argv",
			"-e", "POSIX path of (choose file name with prompt (item 1 of argv) default name (item 2 of argv))",
			"-e", "end run", title, name}
	},
	cancels: []int{1},
}

// winScript shows a Windows Forms dialog and prints the chosen path; when
// the user cancels it prints nothing.
const winScript = `Add-Type -AssemblyName System.Windows.Forms
$d = if ($env:WS_SAVE -eq '1') { New-Object System.Windows.Forms.SaveFileDialog } else { New-Object System.Windows.Forms.OpenFileDialog }
$d.Title = $env:WS_TITLE
if ($env:WS_SAVE -eq '1') { $d.FileName = $env:WS_NAME; $d.OverwritePrompt = $true }
if ($d.ShowDialog() -eq 'OK') { [Console]::Out.Write($d.FileName) }`

// powershell is on every Windows; -STA is needed by the dialogs.
var powershell = tool{
	name: "powershell",
	env: func(title, name string, save bool) []string {
		flag := "0"
		if save {
			flag = "1"
		}
		return []string{"WS_SAVE=" + flag, "WS_TITLE=" + title, "WS_NAME=" + name}
	},
	open: func(title string) []string {
		return []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", winScript}
	},
	save: func(title, name string) []string {
		return []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", winScript}
	},
	cancels: []int{1},
}

// toolsBySystem are the dialog programs a system tries, in order, whatever
// its desktop; a system not listed orders the Linux programs by its desktop
// (preferred).
var toolsBySystem = map[string][]tool{
	"darwin":  slices.Concat([]tool{osascript}, anyDesktop.order),
	"windows": {powershell},
}

// found is an installed dialog program and where it is.
type found struct {
	tool tool
	path string
}

// find is the first dialog program of this system and desktop that is
// installed.
func find() (found, bool) {
	list := installed(preferred(goos, os.Getenv), exec.LookPath)
	if len(list) == 0 {
		return found{}, false
	}
	return list[0], true
}

// Available reports whether a dialog program is installed.
func Available() bool { _, ok := find(); return ok }

func run(ctx context.Context, title, name string, save bool, build func(tool) []string) (string, error) {
	program, ok := find()
	if !ok {
		return "", noProgram()
	}
	cmd := exec.CommandContext(ctx, program.path, build(program.tool)...)
	if program.tool.env != nil {
		cmd.Env = append(os.Environ(), program.tool.env(title, name, save)...)
	}
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && slices.Contains(program.tool.cancels, exit.ExitCode()) {
			return "", ErrCancelled
		}
		return "", err
	}
	chosen := strings.TrimRight(string(out), "\r\n")
	if chosen == "" {
		return "", ErrCancelled
	}
	return chosen, nil
}

// noProgram names the programs that were looked for.
func noProgram() error {
	var names []string
	for _, each := range preferred(goos, os.Getenv) {
		names = append(names, each.name)
	}
	return fmt.Errorf("no file dialog program is installed (looked for %s)", strings.Join(names, ", "))
}

// Open asks for an existing file.
func Open(ctx context.Context, title string) (string, error) {
	return run(ctx, title, "", false, func(program tool) []string { return program.open(title) })
}

// Save asks for a file to write, starting from name.
func Save(ctx context.Context, title, name string) (string, error) {
	return run(ctx, title, name, true, func(program tool) []string { return program.save(title, name) })
}

// System is the file dialog of this desktop.
type System struct{}

// Available implements port.FileDialog.
func (System) Available() bool { return Available() }

// Choose implements port.FileDialog.
func (System) Choose(ctx context.Context, title string, save bool, name string) (string, error) {
	if save {
		return Save(ctx, title, name)
	}
	return Open(ctx, title)
}

var _ port.FileDialog = System{}
