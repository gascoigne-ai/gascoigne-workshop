package statefile

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func sample() state.State {
	return state.State{
		Window:    state.Window{W: 1400, H: 900, X: 40, Y: 30},
		SidebarW:  300,
		PaneSplit: 0.4,
		Expanded:  []string{"Repository", "Repository/Deep"},
		Tabs:      []state.Tab{{Path: "Repository/Search Repos", Pane: 1, Resp: 2}, {Path: "User/Me"}},
		Active:    1,
		Columns:   map[string][]float64{"params": {1, 2, 3}},
	}
}

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "state.json")
	if err := Save(p, sample()); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil || !reflect.DeepEqual(got, sample()) {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err := os.Stat(p + ".tmp"); err == nil {
		t.Fatal("temp file left behind")
	}
}

func TestMissingFileIsEmptyNotAnError(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if err != nil || !reflect.DeepEqual(got, state.State{}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestCorruptFileReportsAndYieldsEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	os.WriteFile(p, []byte("{not json"), 0o644)
	got, err := Load(p)
	if err == nil || !reflect.DeepEqual(got, state.State{}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestUnknownFieldsAreIgnored(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	os.WriteFile(p, []byte(`{"active":2,"future":{"x":1}}`), 0o644)
	got, err := Load(p)
	if err != nil || got.Active != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestEmptyPathIsInMemory(t *testing.T) {
	if err := Save("", sample()); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(""); err != nil {
		t.Fatal(err)
	}
}
