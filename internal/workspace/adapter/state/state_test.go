package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func sample() State {
	return State{
		Window:    Window{W: 1400, H: 900, X: 40, Y: 30},
		SidebarW:  300,
		PaneSplit: 0.4,
		Expanded:  []string{"Repository", "Repository/Deep"},
		Tabs:      []Tab{{Path: "Repository/Search Repos", Pane: 1, Resp: 2}, {Path: "User/Me"}},
		Active:    1,
		Columns:   map[string][]float64{"params": {1, 2, 3}},
	}
}

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "state.json")
	if err := sample().Save(p); err != nil {
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
	if err != nil || !reflect.DeepEqual(got, State{}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestCorruptFileReportsAndYieldsEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	os.WriteFile(p, []byte("{not json"), 0o644)
	got, err := Load(p)
	if err == nil || !reflect.DeepEqual(got, State{}) {
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
	if err := sample().Save(""); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(""); err != nil {
		t.Fatal(err)
	}
}

func TestClampKeepsTheWindowOnTheMonitor(t *testing.T) {
	cases := []struct {
		name string
		in   Window
		want Window
	}{
		{"inside", Window{W: 800, H: 600, X: 100, Y: 100}, Window{W: 800, H: 600, X: 100, Y: 100}},
		{"off to the right", Window{W: 800, H: 600, X: 5000, Y: 100}, Window{W: 800, H: 600, X: 1920 - 800, Y: 100}},
		{"off to the left", Window{W: 800, H: 600, X: -900, Y: -50}, Window{W: 800, H: 600, X: 0, Y: 0}},
		{"bigger than the monitor", Window{W: 4000, H: 3000, X: 0, Y: 0}, Window{W: 1920, H: 1080, X: 0, Y: 0}},
		{"tiny", Window{W: 10, H: 10, X: 0, Y: 0}, Window{W: MinWindowW, H: MinWindowH, X: 0, Y: 0}},
		{"unset stays unset", Window{}, Window{}},
	}
	for _, c := range cases {
		if got := c.in.Clamp(1920, 1080); got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}
