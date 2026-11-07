package state

import "testing"

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
