// Package filedialog asks the desktop for a file name by running the
// dialog program the system has: the native panel through osascript on
// macOS, a PowerShell dialog on Windows, zenity (GNOME and most others) or
// kdialog (KDE) elsewhere. Without one, Available is false and callers fall
// back to asking for the path as text.
package filedialog

import (
	"context"
	"errors"
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
}

// goos is the operating system; tests change it.
var goos = runtime.GOOS

// osascript runs AppleScript, which is on every Mac, and shows the native
// open and save panels. The title and the name travel as arguments of the
// script, never inside its text. Cancelling makes it exit 1.
var osascript = tool{"osascript", nil,
	func(title string) []string {
		return []string{
			"-e", "on run argv",
			"-e", "POSIX path of (choose file with prompt (item 1 of argv))",
			"-e", "end run", title}
	},
	func(title, name string) []string {
		return []string{
			"-e", "on run argv",
			"-e", "POSIX path of (choose file name with prompt (item 1 of argv) default name (item 2 of argv))",
			"-e", "end run", title, name}
	}}

// winScript shows a Windows Forms dialog and prints the chosen path; when
// the user cancels it prints nothing.
const winScript = `Add-Type -AssemblyName System.Windows.Forms
$d = if ($env:WS_SAVE -eq '1') { New-Object System.Windows.Forms.SaveFileDialog } else { New-Object System.Windows.Forms.OpenFileDialog }
$d.Title = $env:WS_TITLE
if ($env:WS_SAVE -eq '1') { $d.FileName = $env:WS_NAME; $d.OverwritePrompt = $true }
if ($d.ShowDialog() -eq 'OK') { [Console]::Out.Write($d.FileName) }`

// powershell is on every Windows; -STA is needed by the dialogs.
var powershell = tool{"powershell",
	func(title, name string, save bool) []string {
		flag := "0"
		if save {
			flag = "1"
		}
		return []string{"WS_SAVE=" + flag, "WS_TITLE=" + title, "WS_NAME=" + name}
	},
	func(title string) []string {
		return []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", winScript}
	},
	func(title, name string) []string {
		return []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", winScript}
	}}

var tools = []tool{
	{"zenity", nil,
		func(title string) []string { return []string{"--file-selection", "--title=" + title} },
		func(title, name string) []string {
			return []string{"--file-selection", "--save", "--confirm-overwrite", "--title=" + title, "--filename=" + name}
		}},
	{"kdialog", nil,
		func(title string) []string { return []string{"--title", title, "--getopenfilename", "."} },
		func(title, name string) []string { return []string{"--title", title, "--getsavefilename", name} }},
}

// toolsBySystem are the dialog programs a system tries, in order; a system
// not listed (Linux, the BSDs) tries tools. A Mac may have zenity from a
// package manager, so it is tried after the native panel.
var toolsBySystem = map[string][]tool{
	"darwin":  slices.Concat([]tool{osascript}, tools),
	"windows": {powershell},
}

// find is the first dialog program on the PATH.
func find() (tool, string, bool) {
	list, ok := toolsBySystem[goos]
	if !ok {
		list = tools
	}
	for _, t := range list {
		if p, err := exec.LookPath(t.name); err == nil {
			return t, p, true
		}
	}
	return tool{}, "", false
}

// Available reports whether a dialog program is installed.
func Available() bool { _, _, ok := find(); return ok }

func run(ctx context.Context, title, name string, save bool, build func(tool) []string) (string, error) {
	t, path, ok := find()
	if !ok {
		return "", errors.New("no file dialog program (osascript, powershell, zenity or kdialog) is installed")
	}
	cmd := exec.CommandContext(ctx, path, build(t)...)
	if t.env != nil {
		cmd.Env = append(os.Environ(), t.env(title, name, save)...)
	}
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 { // both programs exit 1 when the user cancels
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

// Open asks for an existing file.
func Open(ctx context.Context, title string) (string, error) {
	return run(ctx, title, "", false, func(t tool) []string { return t.open(title) })
}

// Save asks for a file to write, starting from name.
func Save(ctx context.Context, title, name string) (string, error) {
	return run(ctx, title, name, true, func(t tool) []string { return t.save(title, name) })
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
