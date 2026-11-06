package history

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func sentAt(minute int, path string) Sent {
	return Sent{At: time.Date(2026, 11, 6, 10, minute, 0, 0, time.UTC), Path: path, Protocol: "rest", Method: "GET", URL: "https://example.test/" + path, Status: 200}
}

func TestSentLogListsNewestFirstAfterReload(t *testing.T) {
	dir := t.TempDir()
	for minute, path := range []string{"a", "b", "c"} {
		if err := AppendSent(dir, sentAt(minute, path)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := LoadSent(dir)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, sent := range got {
		order = append(order, sent.Path)
	}
	if fmt.Sprint(order) != "[c b a]" {
		t.Fatalf("order = %v, want [c b a]", order)
	}
	if got[0].Status != 200 || got[0].URL != "https://example.test/c" {
		t.Fatalf("fields lost: %+v", got[0])
	}
}

func TestSentLogWithoutFileIsEmpty(t *testing.T) {
	got, err := LoadSent(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want nothing", got, err)
	}
}

func TestSentLogKeepsOnlyTheNewestSentLimit(t *testing.T) {
	dir := t.TempDir()
	for minute := 0; minute < SentLimit+SentLimit/2; minute++ {
		if err := AppendSent(dir, sentAt(0, fmt.Sprint(minute))); err != nil {
			t.Fatal(err)
		}
	}
	got, err := LoadSent(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != SentLimit {
		t.Fatalf("len = %d, want %d", len(got), SentLimit)
	}
	if want := fmt.Sprint(SentLimit + SentLimit/2 - 1); got[0].Path != want {
		t.Fatalf("newest = %s, want %s", got[0].Path, want)
	}
	data, err := os.ReadFile(filepath.Join(dir, sentFile))
	if err != nil {
		t.Fatal(err)
	}
	if lines := countLines(data); lines > 2*SentLimit {
		t.Fatalf("file grew to %d lines", lines)
	}
}

func TestSentLogSkipsDamagedLines(t *testing.T) {
	dir := t.TempDir()
	if err := AppendSent(dir, sentAt(1, "a")); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(dir, sentFile), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString("{not json\n")
	file.Close()
	if err := AppendSent(dir, sentAt(2, "b")); err != nil {
		t.Fatal(err)
	}
	got, _ := LoadSent(dir)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func countLines(data []byte) int {
	lines := 0
	for _, char := range data {
		if char == '\n' {
			lines++
		}
	}
	return lines
}
