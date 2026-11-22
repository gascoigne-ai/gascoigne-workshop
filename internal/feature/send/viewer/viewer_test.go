package viewer

import "testing"

func TestFindCount(t *testing.T) {
	tests := []struct {
		query          string
		matches, index int
		want           string
	}{
		{"", 0, -1, ""},
		{"x", 0, -1, "No matches"},
		{"x", 12, 0, "1/12"},
		{"x", 12, 11, "12/12"},
		{"x", 3, -1, "3 matches"},
	}
	for _, test := range tests {
		if got := Count(test.query, test.matches, test.index); got != test.want {
			t.Errorf("Count(%q,%d,%d) = %q, want %q", test.query, test.matches, test.index, got, test.want)
		}
	}
}

func TestStepMovesToTheNextOrPreviousMatchAndWraps(t *testing.T) {
	tests := []struct {
		name                string
		index, dir, matches int
		want                int
	}{
		{"next", 0, 1, 3, 1},
		{"next wraps", 2, 1, 3, 0},
		{"previous", 2, -1, 3, 1},
		{"previous wraps", 0, -1, 3, 2},
		{"first next with none active", -1, 1, 3, 0},
		{"first previous with none active goes to the last", -1, -1, 3, 2},
		{"no matches stays", -1, 1, 0, -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Step(test.index, test.dir, test.matches); got != test.want {
				t.Fatalf("Step(%d, %d, %d) = %d, want %d", test.index, test.dir, test.matches, got, test.want)
			}
		})
	}
}

func TestFlagsOfAViewSettingFallBackToCompact(t *testing.T) {
	tests := []struct {
		id           string
		raw, compact bool
	}{
		{"compact", false, true},
		{"pretty", false, false},
		{"raw", true, false},
		{"unknown", false, true},
		{"", false, true},
	}
	for _, test := range tests {
		if raw, compact := Flags(test.id); raw != test.raw || compact != test.compact {
			t.Errorf("Flags(%q) = %v, %v; want %v, %v", test.id, raw, compact, test.raw, test.compact)
		}
	}
}
