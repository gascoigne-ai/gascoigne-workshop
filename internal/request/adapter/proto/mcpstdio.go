package proto

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

type mcpStdio struct {
	stdin io.WriteCloser
	stop  context.CancelFunc // kills the process; the session owns its life
	mu    sync.Mutex
}

func newMCPStdio(command string, deliver func([]byte), finish func(error)) (*mcpStdio, error) {
	args, err := model.ShellSplit(command)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, errors.New("stdio: needs a command")
	}
	ctx, stop := context.WithCancel(context.Background()) // not Connect's: that one ends with the handshake
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
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
		}
		finish(err)
	}()
	return &mcpStdio{stdin: stdin, stop: stop}, nil
}

func (m *mcpStdio) send(_ context.Context, msg []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.stdin.Write(append(bytes.TrimSpace(msg), '\n'))
	return err
}

func (m *mcpStdio) setVersion(string) {}

func (m *mcpStdio) close(context.Context) error {
	m.stdin.Close()
	time.AfterFunc(time.Second, m.stop) // kills a server that ignores the end of its input; one that is gone is left alone
	return nil
}
