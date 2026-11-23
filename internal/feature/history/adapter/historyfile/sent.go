package historyfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"os"
	"path/filepath"
	"slices"
)

// sentFile is the log inside the history folder: one JSON object per line.
const sentFile = "sent.jsonl"

// sentLine is one line of the log: the version is on every line, since a
// header line would be lost when the log is trimmed.
type sentLine struct {
	Version int `json:"version"`
	history.Sent
}

// sentReaders read a line by version; versions 0 and 1 share a layout.
var sentReaders = []func([]byte) (sentLine, error){storage.Plain[sentLine], storage.Plain[sentLine]}

// sentVersion is the version AppendSent writes.
var sentVersion = len(sentReaders) - 1

// SentLog is the sent-request log kept in the history folder Dir.
type SentLog struct{ Dir string }

// Append adds sent to the log. The file is rewritten to the
// newest history.SentLimit lines once it holds twice that, so it stays small
// without a rewrite on every send.
func (log SentLog) Append(sent history.Sent) error {
	line, err := json.Marshal(sentLine{Version: sentVersion, Sent: sent})
	if err != nil {
		return fmt.Errorf("encode sent request: %w", err)
	}
	if err := os.MkdirAll(log.Dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", log.Dir, err)
	}
	target := filepath.Join(log.Dir, sentFile)
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

// trimSent rewrites the log to its newest history.SentLimit entries when it has
// grown past twice that.
func trimSent(target string) error {
	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read sent log: %w", err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) <= 2*history.SentLimit {
		return nil
	}
	kept := bytes.Join(lines[len(lines)-history.SentLimit:], []byte("\n"))
	if err := storage.WriteFileAtomic(target, append(kept, '\n'), 0o600); err != nil {
		return fmt.Errorf("write sent log: %w", err)
	}
	return nil
}

// Load reads the log, newest first, at most history.SentLimit entries.
// A missing log is no entries; a damaged line is skipped, so one bad write
// does not hide the rest.
func (log SentLog) Load() ([]history.Sent, error) {
	data, err := os.ReadFile(filepath.Join(log.Dir, sentFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sent log: %w", err)
	}
	var all []history.Sent
	for _, line := range bytes.Split(data, []byte("\n")) {
		// A protocol the workshop does not know fails to decode; one left
		// out decodes as none. Either way the line names no request to open.
		if sent, err := decodeSentLine(line); err == nil && sent.Path != "" && sent.Protocol != "" {
			all = append(all, sent)
		}
	}
	slices.Reverse(all)
	if len(all) > history.SentLimit {
		all = all[:history.SentLimit]
	}
	return all, nil
}

// decodeSentLine reads one line of the log with the reader of its version.
func decodeSentLine(line []byte) (history.Sent, error) {
	read, _, err := storage.Read(line, sentReaders)
	return read.Sent, err
}

var _ history.SentStore = SentLog{}
