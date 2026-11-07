package envs

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func collectionWith(names ...string) *model.Collection {
	col := &model.Collection{}
	for _, name := range names {
		col.Envs = append(col.Envs, model.Environment{Name: name})
	}
	return col
}

func TestUniqueNameCountsUpFromTwo(t *testing.T) {
	cases := []struct {
		name   string
		taken  []string
		base   string
		except int
		want   string
	}{
		{"free", []string{"dev"}, "prod", -1, "prod"},
		{"taken", []string{"dev"}, "dev", -1, "dev 2"},
		{"second taken", []string{"dev", "dev 2"}, "dev", -1, "dev 3"},
		{"its own name does not count", []string{"dev", "prod"}, "dev", 0, "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := UniqueName(collectionWith(tc.taken...).Envs, tc.base, tc.except); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSelectClampsToNoneAndReturnsTheNameToRemember(t *testing.T) {
	col := collectionWith("dev", "prod")
	var sel Selection
	if name := sel.Select(col, 1); name != "prod" || sel.Index != 1 {
		t.Fatalf("name %q index %d", name, sel.Index)
	}
	if name := sel.Select(col, 7); name != "" || sel.Index != -1 {
		t.Fatalf("out of range: name %q index %d", name, sel.Index)
	}
}

func TestPickFindsTheRememberedNameOrNone(t *testing.T) {
	col := collectionWith("dev", "prod")
	var sel Selection
	sel.Pick(col, "prod")
	if sel.Index != 1 {
		t.Fatalf("index %d", sel.Index)
	}
	sel.Pick(col, "gone")
	if sel.Index != -1 || sel.Active(col) != nil {
		t.Fatalf("index %d", sel.Index)
	}
}

func TestAddCreatesAUniquelyNamedEnvironmentAndSelectsIt(t *testing.T) {
	col := collectionWith("Environment")
	var sel Selection
	name := sel.Add(col, "")
	if name != "Environment 2" || len(col.Envs) != 2 || sel.Index != 1 || sel.Active(col).Name != "Environment 2" {
		t.Fatalf("name %q envs %+v index %d", name, col.Envs, sel.Index)
	}
}

func TestRenameKeepsTheNameUniqueAndReportsTheRememberedNameOfTheActiveOne(t *testing.T) {
	col := collectionWith("dev", "prod")
	sel := Selection{Index: 1}
	if name, changed := sel.Rename(col, 1, "dev"); !changed || name != "dev 2" {
		t.Fatalf("name %q changed %v", name, changed)
	}
	if _, changed := sel.Rename(col, 0, ""); changed {
		t.Fatal("an empty name renamed")
	}
	if _, changed := sel.Rename(col, 5, "x"); changed {
		t.Fatal("an unknown environment renamed")
	}
}

func TestDeleteMovesTheSelection(t *testing.T) {
	cases := []struct {
		name      string
		index, at int
		want      int
	}{
		{"deleting the active one selects none", 1, 1, -1},
		{"deleting one before it shifts the index", 2, 0, 1},
		{"deleting one after it keeps the index", 0, 2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			col := collectionWith("a", "b", "c")
			sel := Selection{Index: tc.index}
			sel.Delete(col, tc.at)
			if len(col.Envs) != 2 || sel.Index != tc.want {
				t.Fatalf("envs %d index %d, want %d", len(col.Envs), sel.Index, tc.want)
			}
		})
	}
}
