package history

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func entry(status int, body string) Entry {
	return Entry{
		At: time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC), Status: status, StatusText: "OK",
		Duration: 1500 * time.Millisecond, Headers: []model.KV{{Name: "Content-Type", Value: "application/json", Enabled: true}},
		Body: []byte(body),
	}
}

func TestSavedEntriesComeBackAsTheyWere(t *testing.T) {
	dir := t.TempDir()
	want := []Entry{entry(200, `{"a":1}`), entry(404, "not found\x00binary")}
	if err := Save(dir, "API/Users/List", want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir, "API/Users/List")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadingWhatWasNeverSavedIsEmptyNotAnError(t *testing.T) {
	got, err := Load(t.TempDir(), "API/Nothing")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestRequestsKeepSeparateHistories(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "A/one", []Entry{entry(200, "one")}); err != nil {
		t.Fatal(err)
	}
	if err := Save(dir, "A/two", []Entry{entry(201, "two")}); err != nil {
		t.Fatal(err)
	}
	one, _ := Load(dir, "A/one")
	two, _ := Load(dir, "A/two")
	if len(one) != 1 || string(one[0].Body) != "one" || len(two) != 1 || string(two[0].Body) != "two" {
		t.Fatalf("one %v two %v", one, two)
	}
}

func TestSaveKeepsOnlyTheNewestEntriesThatAreNotTooBig(t *testing.T) {
	dir := t.TempDir()
	var entries []Entry
	for range Limit + 5 {
		entries = append(entries, entry(200, "small"))
	}
	entries[0] = entry(200, strings.Repeat("x", MaxBody+1)) // newest, but too big to keep
	if err := Save(dir, "API/Big", entries); err != nil {
		t.Fatal(err)
	}
	got, _ := Load(dir, "API/Big")
	if len(got) != Limit {
		t.Fatalf("kept %d entries, want %d", len(got), Limit)
	}
	for _, kept := range got {
		if len(kept.Body) > MaxBody {
			t.Fatal("an oversize body was written")
		}
	}
}

func TestSavingNothingRemovesTheFile(t *testing.T) {
	dir := t.TempDir()
	_ = Save(dir, "API/x", []Entry{entry(200, "x")})
	if err := Save(dir, "API/x", nil); err != nil {
		t.Fatal(err)
	}
	if files, _ := filepath.Glob(filepath.Join(dir, "*")); len(files) != 0 {
		t.Fatalf("left %v", files)
	}
}

func TestABrokenFileIsAnErrorTheCallerCanReport(t *testing.T) {
	dir := t.TempDir()
	_ = Save(dir, "API/x", []Entry{entry(200, "x")})
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	if err := os.WriteFile(files[0], []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir, "API/x")
	if err == nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestAFileOfAnotherRequestIsNotShown(t *testing.T) {
	dir := t.TempDir()
	_ = Save(dir, "API/x", []Entry{entry(200, "x")})
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	data, _ := os.ReadFile(files[0])
	// Two keys that hash to the same name cannot be built, so the key
	// inside the file is altered instead: the file now claims another request.
	data = []byte(strings.Replace(string(data), "API/x", "API/other", 1))
	if err := os.WriteFile(files[0], data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := Load(dir, "API/x"); len(got) != 0 {
		t.Fatalf("showed %v", got)
	}
}
