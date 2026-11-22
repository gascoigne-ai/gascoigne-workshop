package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/userdir"
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
	response.Sent
}

// sentReaders read a line by version; versions 0 and 1 share a layout.
var sentReaders = []func([]byte) (sentLine, error){fileversion.Plain[sentLine], fileversion.Plain[sentLine]}

// sentVersion is the version AppendSent writes.
var sentVersion = len(sentReaders) - 1

// SentLog is the sent-request log kept in the history folder Dir.
type SentLog struct{ Dir string }

// Append adds sent to the log. The file is rewritten to the
// newest response.SentLimit lines once it holds twice that, so it stays small
// without a rewrite on every send.
func (log SentLog) Append(sent response.Sent) error {
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

// trimSent rewrites the log to its newest response.SentLimit entries when it has
// grown past twice that.
func trimSent(target string) error {
	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read sent log: %w", err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) <= 2*response.SentLimit {
		return nil
	}
	kept := bytes.Join(lines[len(lines)-response.SentLimit:], []byte("\n"))
	if err := userdir.WriteFileAtomic(target, append(kept, '\n'), 0o600); err != nil {
		return fmt.Errorf("write sent log: %w", err)
	}
	return nil
}

// Load reads the log, newest first, at most response.SentLimit entries.
// A missing log is no entries; a damaged line is skipped, so one bad write
// does not hide the rest.
func (log SentLog) Load() ([]response.Sent, error) {
	data, err := os.ReadFile(filepath.Join(log.Dir, sentFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sent log: %w", err)
	}
	var all []response.Sent
	for _, line := range bytes.Split(data, []byte("\n")) {
		// A protocol the workshop does not know fails to decode; one left
		// out decodes as none. Either way the line names no request to open.
		if sent, err := decodeSentLine(line); err == nil && sent.Path != "" && sent.Protocol != "" {
			all = append(all, sent)
		}
	}
	slices.Reverse(all)
	if len(all) > response.SentLimit {
		all = all[:response.SentLimit]
	}
	return all, nil
}

// decodeSentLine reads one line of the log with the reader of its version.
func decodeSentLine(line []byte) (response.Sent, error) {
	read, _, err := fileversion.Read(line, sentReaders)
	return read.Sent, err
}

var _ response.SentStore = SentLog{}
