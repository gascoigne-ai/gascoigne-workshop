package protocol

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
)

// debugFunc is the debug log: slog.Debug in production, a recorder in tests.
type debugFunc func(message string, args ...any)

// A server's complaint is kept for the exit text: the last mcpStderrLines
// lines, each cut to mcpStderrLine bytes, so a chatty or hostile server
// cannot grow it.
const (
	mcpStderrLines = 20
	mcpStderrLine  = 512
)

// tailLines is an io.Writer that keeps the last lines written to it.
type tailLines struct {
	mu       sync.Mutex
	lines    []string // completed lines, oldest first, at most maxLines
	partial  []byte   // the line still being written, at most maxLine bytes
	maxLines int
	maxLine  int
}

func newTailLines(maxLines, maxLine int) *tailLines {
	return &tailLines{maxLines: maxLines, maxLine: maxLine}
}

func (t *tailLines) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, b := range data {
		if b == '\n' {
			t.endLine()
			continue
		}
		if len(t.partial) < t.maxLine { // the rest of an over-long line is dropped
			t.partial = append(t.partial, b)
		}
	}
	return len(data), nil
}

func (t *tailLines) endLine() {
	if len(t.lines) == t.maxLines {
		copy(t.lines, t.lines[1:])
		t.lines = t.lines[:t.maxLines-1]
	}
	t.lines = append(t.lines, string(t.partial))
	t.partial = t.partial[:0]
}

// String is the kept lines, with the unfinished one last.
func (t *tailLines) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	lines := t.lines
	if len(t.partial) > 0 {
		lines = append(slices.Clone(lines), string(t.partial))
	}
	return strings.Join(lines, "\n")
}

type mcpStdio struct {
	stdin io.WriteCloser
	stop  context.CancelFunc // kills the process; the session owns its life
	mu    sync.Mutex
	debug debugFunc
}

func newMCPStdio(command string, deliver func([]byte), finish func(error)) (*mcpStdio, error) {
	args, err := send.ShellSplit(command)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, errors.New("stdio: needs a command")
	}
	ctx, stop := context.WithCancel(context.Background()) // not Connect's: that one ends with the handshake
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stderr := newTailLines(mcpStderrLines, mcpStderrLine)
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		stop()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stop()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		stop()
		return nil, err
	}
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64<<10), mcpMaxMessage)
		for sc.Scan() {
			if line := bytes.TrimSpace(sc.Bytes()); len(line) > 0 {
				deliver(append([]byte(nil), line...))
			}
		}
		scanErr := sc.Err()
		if scanErr != nil { // the reader stopped, so the process would run on unheard
			stop()
		}
		err := cmd.Wait()
		stop()
		switch {
		case scanErr != nil:
			err = fmt.Errorf("read the server's output: %w", scanErr)
		case err == nil:
			err = errors.New("the server exited")
		default:
			err = fmt.Errorf("the server exited: %w", err)
		}
		if tail := stderr.String(); tail != "" {
			err = fmt.Errorf("%w\nstderr:\n%s", err, tail)
		}
		finish(err)
	}()
	return &mcpStdio{stdin: stdin, stop: stop, debug: slog.Debug}, nil
}

func (m *mcpStdio) send(_ context.Context, msg []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.stdin.Write(append(bytes.TrimSpace(msg), '\n'))
	return err
}

func (m *mcpStdio) setVersion(string) {}

func (m *mcpStdio) close(context.Context) error {
	if err := m.stdin.Close(); err != nil { // the process may be gone already
		m.debug("mcp stdio: close the server's input", "error", err)
	}
	time.AfterFunc(time.Second, m.stop) // kills a server that ignores the end of its input; one that is gone is left alone
	return nil
}
