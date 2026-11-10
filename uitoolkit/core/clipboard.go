package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

// Clipboard reads and writes the system clipboard as text.
type Clipboard interface {
	Read() (string, error)
	Write(text string) error
}

// clipboardTool is a pair of commands that read and write the clipboard.
type clipboardTool struct {
	read, write []string
}

// clipboardTools are tried in order; the first whose programs are both
// installed wins. Ebitengine has no clipboard API, so the toolkit uses
// the platform's own tools instead of a dependency.
var clipboardTools = []clipboardTool{
	{read: []string{"wl-paste", "--no-newline"}, write: []string{"wl-copy"}},
	{read: []string{"xclip", "-selection", "clipboard", "-o"}, write: []string{"xclip", "-selection", "clipboard", "-i"}},
	{read: []string{"xsel", "--clipboard", "--output"}, write: []string{"xsel", "--clipboard", "--input"}},
	{read: []string{"pbpaste"}, write: []string{"pbcopy"}},
}

const clipboardTimeout = 2 * time.Second

// SystemClipboard returns the clipboard backed by the first available
// tool. When none is installed, every call fails with an error that
// says what to install.
func SystemClipboard() Clipboard {
	return pickClipboard(exec.LookPath)
}

func pickClipboard(lookPath func(string) (string, error)) Clipboard {
	for _, t := range clipboardTools {
		if _, err := lookPath(t.read[0]); err != nil {
			continue
		}
		if _, err := lookPath(t.write[0]); err != nil {
			continue
		}
		return cmdClipboard{tool: t}
	}
	return noClipboard{}
}

type cmdClipboard struct{ tool clipboardTool }

func (c cmdClipboard) Read() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.tool.read[0], c.tool.read[1:]...).Output()
	if err != nil {
		return "", fmt.Errorf("read clipboard with %s: %w", c.tool.read[0], err)
	}
	return string(out), nil
}

func (c cmdClipboard) Write(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.tool.write[0], c.tool.write[1:]...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write clipboard with %s: %w", c.tool.write[0], err)
	}
	return nil
}

type noClipboard struct{}

var errNoClipboard = errors.New("no clipboard tool found (install wl-clipboard or xclip)")

func (noClipboard) Read() (string, error) { return "", errNoClipboard }
func (noClipboard) Write(string) error    { return errNoClipboard }

// ReportClipboard wraps inner so that every outcome is reported: what was
// copied (isErr false), or why the clipboard is not usable, for example
// when no tool is installed (isErr true). A successful read is silent.
func ReportClipboard(inner Clipboard, report func(message string, isErr bool)) Clipboard {
	return reportClipboard{inner: inner, report: report}
}

type reportClipboard struct {
	inner  Clipboard
	report func(message string, isErr bool)
}

func (clipboard reportClipboard) Read() (string, error) {
	text, err := clipboard.inner.Read()
	if err != nil {
		clipboard.report(err.Error(), true)
	}
	return text, err
}

func (clipboard reportClipboard) Write(text string) error {
	if err := clipboard.inner.Write(text); err != nil {
		clipboard.report(err.Error(), true)
		return err
	}
	clipboard.report(fmt.Sprintf("Copied %d characters", utf8.RuneCountInString(text)), false)
	return nil
}
