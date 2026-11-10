package viewer

import "testing"

func TestNewTabFollowsTheSettingWithNoActiveMatch(t *testing.T) {
	tests := []struct {
		setting string
		want    Mode
	}{
		{"compact", Compact}, {"pretty", Pretty}, {"raw", Raw}, {"", Compact}, {"nonsense", Compact},
	}
	for _, test := range tests {
		if got := New(test.setting); got.Mode != test.want || got.Match != -1 || got.FindOpen {
			t.Errorf("New(%q) = %+v, want mode %v", test.setting, got, test.want)
		}
	}
}

func TestModeChosenChangesOnlyTheMode(t *testing.T) {
	before := Model{Mode: Compact, FindOpen: true, Match: 2}
	if got := Update(before, ModeChosen{Mode: Raw}); got != (Model{Mode: Raw, FindOpen: true, Match: 2}) {
		t.Fatalf("%+v", got)
	}
}

func TestFindOpensAndClosesDroppingTheMatch(t *testing.T) {
	opened := Update(New("pretty"), FindOpened{})
	if !opened.FindOpen {
		t.Fatal("find bar must open")
	}
	closed := Update(Model{FindOpen: true, Match: 3}, FindClosed{})
	if closed.FindOpen || closed.Match != -1 {
		t.Fatalf("%+v", closed)
	}
}

func TestSearchedActivatesTheFirstMatchOrNone(t *testing.T) {
	if got := Update(Model{Match: 4}, Searched{Matches: 2}); got.Match != 0 {
		t.Fatalf("found: %+v", got)
	}
	if got := Update(Model{Match: 4}, Searched{Matches: 0}); got.Match != -1 {
		t.Fatalf("none: %+v", got)
	}
}

func TestSteppedWrapsAndStaysWithoutMatches(t *testing.T) {
	tests := []struct {
		name  string
		from  int
		step  Stepped
		match int
	}{
		{"next", 0, Stepped{1, 3}, 1},
		{"wraps forward", 2, Stepped{1, 3}, 0},
		{"wraps back", 0, Stepped{-1, 3}, 2},
		{"back from none goes to the last", -1, Stepped{-1, 3}, 2},
		{"no matches", -1, Stepped{1, 0}, -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Update(Model{Match: test.from}, test.step); got.Match != test.match {
				t.Fatalf("Match = %d, want %d", got.Match, test.match)
			}
		})
	}
}
