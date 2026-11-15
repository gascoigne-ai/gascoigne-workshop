package state

import "testing"

// A state file written when one collection was open at a time names no
// collection: its tabs and folders belong to the one open now.
func TestCollectionAt(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		dirs []string
		want int
	}{
		{"a saved collection by its folder", "/b", []string{"/a", "/b"}, 1},
		{"a folder no longer open", "/c", []string{"/a", "/b"}, -1},
		{"none open", "", nil, -1},
		{"the one not saved", "", []string{"/a", ""}, 1},
		{"an older file: the first", "", []string{"/a", "/b"}, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CollectionAt(test.ref, test.dirs); got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
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
