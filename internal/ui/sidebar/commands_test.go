package sidebar

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
)

// closer records the collection a command asked to close.
type closer struct {
	noop
	closed []*request.Collection
}

func (closer *closer) Close(col *request.Collection) { closer.closed = append(closer.closed, col) }

func TestTheSidebarOffersCollectionCommandsWithTheirDefaultKeys(t *testing.T) {
	_, side, _, _ := withShop(t)
	var ids []keymap.ID
	defaults := map[keymap.ID]keymap.Chord{
		"open-workspace":   {Key: int(ebiten.KeyO), Ctrl: true},
		"close-collection": {Key: int(ebiten.KeyW), Ctrl: true, Shift: true},
	}
	for _, command := range side.Commands() {
		ids = append(ids, command.ID)
		if command.Default != defaults[command.ID] || command.Name == "" || command.Run == nil {
			t.Errorf("%s: default %+v name %q", command.ID, command.Default, command.Name)
		}
	}
	if want := []keymap.ID{"open-workspace", "close-collection"}; !slices.Equal(ids, want) {
		t.Fatalf("ids %v, want %v", ids, want)
	}
}

func TestTheSidebarCommandsAskTheHost(t *testing.T) {
	_, side, fake, col := withShop(t)
	closing := &closer{}
	side.deps.Collection = closing
	for _, command := range side.Commands() {
		command.Run()
	}
	if fake.created != 1 {
		t.Errorf("open-workspace asked %d times, want 1", fake.created)
	}
	if len(closing.closed) != 1 || closing.closed[0] != col {
		t.Errorf("close-collection closed %v, want the focused collection", closing.closed)
	}
}

func TestCloseCollectionDoesNothingWithoutAFocusedCollection(t *testing.T) {
	fake := &fake{}
	_, side := newSidebar(t, fake)
	closing := &closer{}
	side.deps.Collection = closing
	for _, command := range side.Commands() {
		if command.ID == "close-collection" {
			command.Run()
		}
	}
	if len(closing.closed) != 0 {
		t.Fatalf("closed %v with no collection open", closing.closed)
	}
}
