package tabstrip

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
)

func commandByID(t *testing.T, commands []keymap.Command, id keymap.ID) keymap.Command {
	t.Helper()
	for _, command := range commands {
		if command.ID == id {
			return command
		}
	}
	t.Fatalf("no command %q in %d commands", id, len(commands))
	return keymap.Command{}
}

func TestTheStripOffersTabCommandsWithTheirDefaultKeys(t *testing.T) {
	world := twoTabs(t)
	commands := world.strip.Commands()
	var ids []keymap.ID
	for _, command := range commands {
		ids = append(ids, command.ID)
	}
	if want := []keymap.ID{"new-request", "close-tab", "next-tab", "prev-tab"}; !slices.Equal(ids, want) {
		t.Fatalf("ids %v, want %v", ids, want)
	}
	defaults := map[keymap.ID]keymap.Chord{
		"new-request": {Key: int(ebiten.KeyT), Ctrl: true},
		"close-tab":   {Key: int(ebiten.KeyW), Ctrl: true},
		"next-tab":    {Key: int(ebiten.KeyTab), Ctrl: true},
		"prev-tab":    {Key: int(ebiten.KeyTab), Ctrl: true, Shift: true},
	}
	for _, command := range commands {
		if command.Default != defaults[command.ID] || command.Name == "" {
			t.Errorf("%s: default %+v name %q, want %+v", command.ID, command.Default, command.Name, defaults[command.ID])
		}
	}
}

func TestTheTabCommandsAskTheHost(t *testing.T) {
	world := twoTabs(t)
	commands := world.strip.Commands()
	commandByID(t, commands, "prev-tab").Run()
	if got := world.fake.set.Active(); got != 0 {
		t.Fatalf("prev-tab left tab %d active, want 0", got)
	}
	commandByID(t, commands, "next-tab").Run()
	if got := world.fake.set.Active(); got != 1 {
		t.Fatalf("next-tab left tab %d active, want 1", got)
	}
	commandByID(t, commands, "new-request").Run()
	if world.fake.newCount != 1 || len(world.fake.set.Tabs()) != 3 {
		t.Fatalf("new-request made %d requests, %d tabs", world.fake.newCount, len(world.fake.set.Tabs()))
	}
	commandByID(t, commands, "close-tab").Run()
	if len(world.fake.set.Tabs()) != 2 {
		t.Fatalf("close-tab left %d tabs, want 2", len(world.fake.set.Tabs()))
	}
}
