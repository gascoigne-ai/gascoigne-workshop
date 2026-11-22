package history

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func entry(status int, body string) response.Entry {
	return response.Entry{
		At: time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC), Status: status, StatusText: "OK",
		Duration: 1500 * time.Millisecond, Headers: []request.KV{{Name: "Content-Type", Value: "application/json", Enabled: true}},
		Body: []byte(body),
	}
}

func TestSavedEntriesComeBackAsTheyWere(t *testing.T) {
	dir := t.TempDir()
	want := []response.Entry{entry(200, `{"a":1}`), entry(404, "not found\x00binary")}
	if err := Dir(dir).Save("API/Users/List", want); err != nil {
		t.Fatal(err)
	}
	got, err := Dir(dir).Load("API/Users/List")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadingWhatWasNeverSavedIsEmptyNotAnError(t *testing.T) {
	got, err := Dir(t.TempDir()).Load("API/Nothing")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestRequestsKeepSeparateHistories(t *testing.T) {
	dir := t.TempDir()
	if err := Dir(dir).Save("A/one", []response.Entry{entry(200, "one")}); err != nil {
		t.Fatal(err)
	}
	if err := Dir(dir).Save("A/two", []response.Entry{entry(201, "two")}); err != nil {
		t.Fatal(err)
	}
	one, _ := Dir(dir).Load("A/one")
	two, _ := Dir(dir).Load("A/two")
	if len(one) != 1 || string(one[0].Body) != "one" || len(two) != 1 || string(two[0].Body) != "two" {
		t.Fatalf("one %v two %v", one, two)
	}
}

func TestSaveKeepsOnlyTheNewestEntriesThatAreNotTooBig(t *testing.T) {
	dir := t.TempDir()
	var entries []response.Entry
	for range Limit + 5 {
		entries = append(entries, entry(200, "small"))
	}
	entries[0] = entry(200, strings.Repeat("x", MaxBody+1)) // newest, but too big to keep
	if err := Dir(dir).Save("API/Big", entries); err != nil {
		t.Fatal(err)
	}
	got, _ := Dir(dir).Load("API/Big")
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
	_ = Dir(dir).Save("API/x", []response.Entry{entry(200, "x")})
	if err := Dir(dir).Save("API/x", nil); err != nil {
		t.Fatal(err)
	}
	if files, _ := filepath.Glob(filepath.Join(dir, "*")); len(files) != 0 {
		t.Fatalf("left %v", files)
	}
}

func TestABrokenFileIsAnErrorTheCallerCanReport(t *testing.T) {
	dir := t.TempDir()
	_ = Dir(dir).Save("API/x", []response.Entry{entry(200, "x")})
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	if err := os.WriteFile(files[0], []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Dir(dir).Load("API/x")
	if err == nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestAFileOfAnotherRequestIsNotShown(t *testing.T) {
	dir := t.TempDir()
	_ = Dir(dir).Save("API/x", []response.Entry{entry(200, "x")})
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	data, _ := os.ReadFile(files[0])
	// Two keys that hash to the same name cannot be built, so the key
	// inside the file is altered instead: the file now claims another request.
	data = []byte(strings.Replace(string(data), "API/x", "API/other", 1))
	if err := os.WriteFile(files[0], data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := Dir(dir).Load("API/x"); len(got) != 0 {
		t.Fatalf("showed %v", got)
	}
}
