package envs

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func TestMenuTicksTheSelectionAndDisablesRenameAndDeleteWithoutOne(t *testing.T) {
	col := &request.Collection{Envs: []request.Environment{{Name: "Dev"}, {Name: "Prod"}}}
	rows := Menu(Selection{Index: 1}, col)
	labels := []string{"No environment", "Dev", "Prod", "", "New environment...", "Rename...", "Delete"}
	if len(rows) != len(labels) {
		t.Fatalf("%d rows: %+v", len(rows), rows)
	}
	for i, want := range labels {
		if rows[i].Label != want {
			t.Errorf("row %d = %q, want %q", i, rows[i].Label, want)
		}
	}
	for i, want := range []bool{false, false, true} {
		if rows[i].Checked != want {
			t.Errorf("row %d checked = %v, want %v", i, rows[i].Checked, want)
		}
	}
	if rows[3].Action != Separator || rows[5].Disabled || rows[6].Disabled || rows[5].Env != 1 || rows[2].Env != 1 {
		t.Fatalf("%+v", rows)
	}
	none := Menu(None(), col)
	if none[0].Label != "No environment" || !none[0].Checked || !none[5].Disabled || !none[6].Disabled {
		t.Fatalf("%+v", none)
	}
}
