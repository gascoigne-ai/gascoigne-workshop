package request

import "testing"

func TestEnvIndexFindsTheEnvironmentByName(t *testing.T) {
	envs := func(names ...string) Collection {
		col := Collection{}
		for _, name := range names {
			col.Envs = append(col.Envs, Environment{Name: name})
		}
		return col
	}
	tests := []struct {
		name string
		col  Collection
		look string
		want int
	}{
		{"no environments", envs(), "Dev", -1},
		{"a match", envs("Dev", "Prod"), "Prod", 1},
		{"an unknown name", envs("Dev"), "Prod", -1},
		{"the last of equal names", envs("Dev", "Prod", "Dev"), "Dev", 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.col.EnvIndex(test.look); got != test.want {
				t.Errorf("EnvIndex(%q) = %d, want %d", test.look, got, test.want)
			}
		})
	}
}
