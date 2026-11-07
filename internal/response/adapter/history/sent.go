package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"os"
	"path/filepath"
	"slices"
)

// sentFile is the log inside the history folder: one JSON object per line.
const sentFile = "sent.jsonl"

// AppendSent adds sent to the log in dir. The file is rewritten to the
// newest port.SentLimit lines once it holds twice that, so it stays small
// without a rewrite on every send.
func AppendSent(dir string, sent port.Sent) error {
	line, err := json.Marshal(sent)
	if err != nil {
		return fmt.Errorf("encode sent request: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	target := filepath.Join(dir, sentFile)
	file, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open sent log: %w", err)
	}
	_, err = file.Write(append(line, '\n'))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write sent log: %w", err)
	}
	return trimSent(target)
}

// trimSent rewrites the log to its newest port.SentLimit entries when it has
// grown past twice that.
func trimSent(target string) error {
	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read sent log: %w", err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) <= 2*port.SentLimit {
		return nil
	}
	kept := bytes.Join(lines[len(lines)-port.SentLimit:], []byte("\n"))
	temp := target + ".tmp"
	if err := os.WriteFile(temp, append(kept, '\n'), 0o600); err != nil {
		return fmt.Errorf("write sent log: %w", err)
	}
	if err := os.Rename(temp, target); err != nil {
		return fmt.Errorf("replace sent log: %w", err)
	}
	return nil
}

// LoadSent reads the log of dir, newest first, at most port.SentLimit entries.
// A missing log is no entries; a damaged line is skipped, so one bad write
// does not hide the rest.
func LoadSent(dir string) ([]port.Sent, error) {
	data, err := os.ReadFile(filepath.Join(dir, sentFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sent log: %w", err)
	}
	var all []port.Sent
	for _, line := range bytes.Split(data, []byte("\n")) {
		var sent port.Sent
		if json.Unmarshal(line, &sent) == nil && sent.Path != "" {
			all = append(all, sent)
		}
	}
	slices.Reverse(all)
	if len(all) > port.SentLimit {
		all = all[:port.SentLimit]
	}
	return all, nil
}

// SentLog is the sent-request log kept in the history folder Dir.
type SentLog struct{ Dir string }

// Append implements port.SentStore.
func (log SentLog) Append(sent port.Sent) error { return AppendSent(log.Dir, sent) }

// Load implements port.SentStore.
func (log SentLog) Load() ([]port.Sent, error) { return LoadSent(log.Dir) }

var _ port.SentStore = SentLog{}
