package statefile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
)

// The fixtures are state.json files written by the program: v0 before
// versions existed, v1 by this build over the same folder a run later. The
// sandbox folder is replaced by /tmp/workshop-fixture.
const fixtureRoot = "/tmp/workshop-fixture/collection"

func TestLoadReadsStateFixtures(t *testing.T) {
	window := state.Window{W: 1280, H: 800, X: 95, Y: 66}
	tab := func(path, pane string) state.Tab {
		return state.Tab{Collection: fixtureRoot, Path: path, PaneLabel: pane}
	}
	tests := []struct {
		file string
		want state.State
	}{
		{"state-v0.json", state.State{Window: window, Active: 1, Tabs: []state.Tab{
			tab("Index", "Params"), tab("Echo", "Params"), tab("Status 418", "Params"), tab("MCP Tools", "Headers"),
		}}},
		{"state-v1.json", state.State{Window: window, Active: 2, Tabs: []state.Tab{
			tab("Index", "Params"), tab("Status 418", "Params"), tab("Echo", "Params"),
		}}},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			got, err := disk.Load(filepath.Join("testdata", test.file))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Load = %+v\nwant %+v", got, test.want)
			}
		})
	}
}

func TestSaveLoadRoundTripsV1Fixture(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "state-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := disk.Load(filepath.Join("testdata", "state-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err := disk.Save(path, loaded); err != nil {
		t.Fatal(err)
	}
	if saved, _ := os.ReadFile(path); !bytes.Equal(saved, fixture) {
		t.Fatalf("saved\n%s\nwant the fixture\n%s", saved, fixture)
	}
}

func TestSaveWritesCurrentVersionFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := disk.Save(path, sample()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"version\": 1,\n"; !bytes.HasPrefix(data, []byte(want)) {
		t.Fatalf("state.json starts %q, want %q", data[:min(len(data), 40)], want)
	}
}

// withVersion is the v1 fixture with its version replaced by raw, the way
// a file of another build would carry it: the bad versions are not fixture
// files.
func withVersion(t *testing.T, raw string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "state-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Replace(data, []byte(`"version": 1`), []byte(`"version": `+raw), 1)
}

func TestLoadRejectsBadVersions(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		wantNewer bool
	}{
		{"newer", "2", true},
		{"negative", "-1", false},
		{"fractional", "1.5", false},
		{"text", `"1"`, false},
		{"null", "null", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			if err := os.WriteFile(path, withVersion(t, test.version), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := disk.Load(path)
			if err == nil || !reflect.DeepEqual(got, state.State{}) {
				t.Fatalf("Load = %+v, %v; want the empty state and an error", got, err)
			}
			var newer *storage.NewerError
			if errors.As(err, &newer) != test.wantNewer {
				t.Fatalf("error %v: NewerError = %v, want %v", err, !test.wantNewer, test.wantNewer)
			}
			if !strings.Contains(err.Error(), "state.json") {
				t.Fatalf("error %q does not name the file", err)
			}
		})
	}
}
