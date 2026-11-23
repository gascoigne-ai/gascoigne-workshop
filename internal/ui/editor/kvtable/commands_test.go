package kvtable

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
)

func TestTheTablesOfferRowCommandsWithTheirDefaultKeys(t *testing.T) {
	world := newWorld(t)
	var ids []keymap.ID
	defaults := map[keymap.ID]keymap.Chord{
		"duplicate-row": {Key: int(ebiten.KeyD), Ctrl: true},
		"move-row-up":   {Key: int(ebiten.KeyArrowUp), Alt: true},
		"move-row-down": {Key: int(ebiten.KeyArrowDown), Alt: true},
	}
	for _, command := range world.kv.Commands() {
		ids = append(ids, command.ID)
		if command.Default != defaults[command.ID] || command.Name == "" || command.Run == nil {
			t.Errorf("%s: default %+v name %q", command.ID, command.Default, command.Name)
		}
	}
	if want := []keymap.ID{"duplicate-row", "move-row-up", "move-row-down"}; !slices.Equal(ids, want) {
		t.Fatalf("ids %v, want %v", ids, want)
	}
}

func TestTheRowCommandsChangeTheSelectedRow(t *testing.T) {
	world := newWorld(t)
	byID := map[keymap.ID]func(){}
	for _, command := range world.kv.Commands() {
		byID[command.ID] = command.Run
	}
	world.kv.Select(world.params, 1)
	byID["duplicate-row"]()
	if rows := world.tab.Req.Params; len(rows) != 4 || rows[1].Name != "order" || rows[2].Name != "order" {
		t.Fatalf("duplicate-row: %+v", rows)
	}
	byID["move-row-up"]()
	byID["move-row-down"]()
	if rows := world.tab.Req.Params; len(rows) != 4 || rows[2].Name != "order" {
		t.Fatalf("move up then down must leave the rows: %+v", rows)
	}
}
