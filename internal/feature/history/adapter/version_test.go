package history

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/fileversion"
)

// The fixtures are files the program wrote for the unsaved sample's
// "Status 418" request: history-v0.json before files carried a version,
// history-v1.json by this build a run later. sent-mixed.jsonl is the sent
// log of the same runs: six v0 lines, then four v1 lines appended to them
// (the sandbox folder replaced by /tmp/workshop-fixture).
const fixtureKey = "Status 418"

// placeFixture puts the history fixture name where Load finds the history
// of key in a fresh folder, with edit applied to its bytes.
func placeFixture(t *testing.T, name, key string, edit func([]byte) []byte) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(path(dir, key), edit(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func unchanged(data []byte) []byte { return data }

func TestLoadReadsHistoryFixtures(t *testing.T) {
	tests := []struct {
		file    string
		entries int
	}{
		{"history-v0.json", 2},
		{"history-v1.json", 3},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) { checkFixtureHistory(t, test.file, test.entries) })
	}
}

// checkFixtureHistory loads the history fixture file and checks it holds
// count teapot responses.
func checkFixtureHistory(t *testing.T, file string, count int) {
	t.Helper()
	dir := placeFixture(t, file, fixtureKey, unchanged)
	entries, err := Dir(dir).Load(fixtureKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != count {
		t.Fatalf("got %d entries, want %d", len(entries), count)
	}
	first := entries[0]
	if first.Status != 418 || first.StatusText != "I'm a teapot" || first.At.IsZero() || first.Duration <= 0 {
		t.Errorf("first entry = %+v", first)
	}
	if len(first.Headers) == 0 || first.Headers[0].Name != "Content-Length" {
		t.Errorf("headers = %+v", first.Headers)
	}
	if !bytes.Contains(first.Body, []byte("teapot")) {
		t.Errorf("body = %q", first.Body)
	}
}

func TestSaveWritesVersion(t *testing.T) {
	dir := t.TempDir()
	if err := Dir(dir).Save("a", []history.Entry{entry(200, "ok")}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path(dir, "a"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"version":1,"Key":"a","Entries":[{"At":`; !bytes.HasPrefix(data, []byte(want)) {
		t.Fatalf("file starts %q, want %q", data[:min(len(data), 50)], want)
	}
}

func TestSaveLoadRoundTripsV1(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "history-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := placeFixture(t, "history-v1.json", fixtureKey, unchanged)
	entries, err := Dir(dir).Load(fixtureKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := Dir(dir).Save(fixtureKey, entries); err != nil {
		t.Fatal(err)
	}
	if saved, _ := os.ReadFile(path(dir, fixtureKey)); !bytes.Equal(saved, fixture) {
		t.Fatalf("saved\n%s\nwant the fixture\n%s", saved, fixture)
	}
}

// withVersion replaces the version 1 of a line or file by raw, the way one
// of another build would carry it: the bad versions are not fixture files.
func withVersion(raw string) func([]byte) []byte {
	return func(data []byte) []byte {
		return bytes.Replace(data, []byte(`"version":1`), []byte(`"version":`+raw), 1)
	}
}

func TestLoadRejectsBadVersions(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		key       string // the key Load asks for
		wantNewer bool
	}{
		{"newer", "2", fixtureKey, true},
		{"newer, before the key is compared", "2", "Another", true},
		{"negative", "-1", fixtureKey, false},
		{"text", `"1"`, fixtureKey, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "history-v1.json"))
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			// the file sits where test.key looks, holding the fixture's own key
			if err := os.WriteFile(path(dir, test.key), withVersion(test.version)(data), 0o600); err != nil {
				t.Fatal(err)
			}
			entries, err := Dir(dir).Load(test.key)
			if err == nil || entries != nil {
				t.Fatalf("Load = %d entries, %v; want none and an error", len(entries), err)
			}
			var newer *fileversion.NewerError
			if errors.As(err, &newer) != test.wantNewer {
				t.Fatalf("error %v: is NewerError = %v, want %v", err, !test.wantNewer, test.wantNewer)
			}
		})
	}
}

// sentFixtureLines are the lines of a sent-log fixture.
func sentFixtureLines(t *testing.T, name string) [][]byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Split(bytes.TrimSpace(data), []byte("\n"))
}

func TestDecodeSentLineByVersion(t *testing.T) {
	lines := sentFixtureLines(t, "sent-mixed.jsonl")
	v0, v1 := lines[5], lines[9] // both sent from the saved collection
	tests := []struct {
		name      string
		line      []byte
		wantPath  string
		wantErr   bool
		wantNewer bool
	}{
		{name: "v0", line: v0, wantPath: "Echo"},
		{name: "v1", line: v1, wantPath: "Echo"},
		{name: "newer", line: withVersion("2")(v1), wantErr: true, wantNewer: true},
		{name: "negative", line: withVersion("-1")(v1), wantErr: true},
		{name: "text", line: withVersion(`"1"`)(v1), wantErr: true},
		{name: "empty", line: nil, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sent, err := decodeSentLine(test.line)
			if test.wantErr {
				var newer *fileversion.NewerError
				if err == nil || errors.As(err, &newer) != test.wantNewer {
					t.Fatalf("decodeSentLine(%s) = %+v, %v; want an error, NewerError %v", test.line, sent, err, test.wantNewer)
				}
				return
			}
			if err != nil || sent.Path != test.wantPath || sent.Collection != "/tmp/workshop-fixture/collection" || sent.Status != 200 {
				t.Fatalf("decodeSentLine(%s) = %+v, %v", test.line, sent, err)
			}
		})
	}
}

func TestLoadSentReadsMixedVersionLog(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "sent-mixed.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, sentFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Dir(dir).Sent().Load()
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, sent := range got {
		paths = append(paths, sent.Path)
	}
	want := "[Echo Status 418 Status 418 Index Echo Status 418 Status 418 Echo Echo Index]"
	if fmt.Sprint(paths) != want {
		t.Fatalf("LoadSent paths = %v, want %s (newest first, both versions)", paths, want)
	}
}

func TestLoadSentSkipsNewerLines(t *testing.T) {
	lines := sentFixtureLines(t, "sent-mixed.jsonl")
	lines[9] = withVersion("2")(lines[9])
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, sentFile), append(bytes.Join(lines, []byte("\n")), '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Dir(dir).Sent().Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 9 || got[0].Path != "Status 418" || got[0].Collection != "/tmp/workshop-fixture/collection" {
		t.Fatalf("LoadSent = %+v; want the nine other lines, newest first", got)
	}
}

func TestAppendSentWritesVersionOnEveryLine(t *testing.T) {
	dir := t.TempDir()
	for minute, path := range []string{"a", "b"} {
		if err := Dir(dir).Sent().Append(sentAt(minute, path)); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, sentFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if !strings.HasPrefix(line, `{"version":1,"At":`) {
			t.Errorf("line %s does not start with the version", line)
		}
	}
}

// A line of a newer build is not parsed when the log is trimmed: it stays
// until it is among the oldest.
func TestTrimSentKeepsNewerLines(t *testing.T) {
	dir := t.TempDir()
	var log bytes.Buffer
	for range 2*history.SentLimit - 1 {
		log.WriteString(`{"version":1,"At":"2026-11-06T10:00:00Z","Path":"x","Protocol":"rest"}` + "\n")
	}
	newer := `{"version":2,"Path":"from a newer build"}`
	log.WriteString(newer + "\n")
	if err := os.WriteFile(filepath.Join(dir, sentFile), log.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Dir(dir).Sent().Append(sentAt(0, "y")); err != nil { // one past the limit: trims
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, sentFile))
	if err != nil {
		t.Fatal(err)
	}
	if lines := countLines(data); lines != history.SentLimit {
		t.Fatalf("the log has %d lines, want it trimmed to %d", lines, history.SentLimit)
	}
	if !strings.Contains(string(data), newer+"\n") {
		t.Fatalf("the trim dropped the newer line")
	}
}
