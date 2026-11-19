package severity

import "testing"

func TestLevelNamesItself(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{Info, "info"},
		{Error, "error"},
		{Level(200), "level(200)"},
	}
	for _, test := range tests {
		if got := test.level.String(); got != test.want {
			t.Errorf("Level(%d).String() = %q, want %q", uint8(test.level), got, test.want)
		}
	}
}

func TestAllListsEveryLevelOnce(t *testing.T) {
	if got := All(); len(got) != 2 || got[0] != Info || got[1] != Error {
		t.Fatalf("All() = %v", got)
	}
}
