package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fragment groups children without adding a box, so For can return many
// nodes where one is expected. Parents splice it away.
type fragment struct{ plain }

func (fragment) draw(*El, *widget.Ctx, *ebiten.Image) {}

func isFragment(e *El) bool {
	_, ok := e.b.(fragment)
	return ok
}

// Frag returns items as one value, like a Vue fragment.
func Frag(items ...Item) *El { return newEl(fragment{}, items) }

// For is v-for: one node per item.
func For[T any](items []T, build func(i int, item T) Node) Node {
	nodes := make([]Item, 0, len(items))
	for i, it := range items {
		nodes = append(nodes, build(i, it))
	}
	return Frag(nodes...)
}
