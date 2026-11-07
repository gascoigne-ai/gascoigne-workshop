// Package filedialog asks the desktop for a file name by running the
// dialog program the system has: zenity (GNOME and most others) or
// kdialog (KDE). Without either, Available is false and callers fall back
// to asking for the path as text.
package filedialog

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
)

// ErrCancelled is returned when the user closed the dialog.
var ErrCancelled = port.ErrCancelled

// tool describes how to run one dialog program.
type tool struct {
	name string
	open func(title string) []string
	save func(title, name string) []string
}

var tools = []tool{
	{"zenity",
		func(title string) []string { return []string{"--file-selection", "--title=" + title} },
		func(title, name string) []string {
			return []string{"--file-selection", "--save", "--confirm-overwrite", "--title=" + title, "--filename=" + name}
		}},
	{"kdialog",
		func(title string) []string { return []string{"--title", title, "--getopenfilename", "."} },
		func(title, name string) []string { return []string{"--title", title, "--getsavefilename", name} }},
}

// find is the first dialog program on the PATH.
func find() (tool, string, bool) {
	for _, t := range tools {
		if p, err := exec.LookPath(t.name); err == nil {
			return t, p, true
		}
	}
	return tool{}, "", false
}

// Available reports whether a dialog program is installed.
func Available() bool { _, _, ok := find(); return ok }

func run(ctx context.Context, build func(tool) []string) (string, error) {
	t, path, ok := find()
	if !ok {
		return "", errors.New("no file dialog program (zenity or kdialog) is installed")
	}
	out, err := exec.CommandContext(ctx, path, build(t)...).Output()
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
	return run(ctx, func(t tool) []string { return t.open(title) })
}

// Save asks for a file to write, starting from name.
func Save(ctx context.Context, title, name string) (string, error) {
	return run(ctx, func(t tool) []string { return t.save(title, name) })
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
