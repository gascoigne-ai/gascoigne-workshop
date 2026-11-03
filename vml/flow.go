package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// fragment groups children without adding a box, so For can return many
// nodes where one is expected. Parents splice it away.
type fragment struct{ plain }

func (fragment) draw(*El, *widget.Ctx, *ebiten.Image) {}

func isFragment(e *El) bool {
	_, ok := e.b.(fragment)
	return ok
}

// Frag returns nodes as one value, like a Vue fragment.
func Frag(nodes ...Node) *El { return newEl(fragment{}, nodes) }

// If is Vue's v-if: the node when cond holds, else nothing.
func If(cond bool, n Node) Node {
	if cond {
		return n
	}
	return nil
}

// Either is v-if with v-else.
func Either(cond bool, yes, no Node) Node {
	if cond {
		return yes
	}
	return no
}

// For is v-for: one node per item.
func For[T any](items []T, build func(i int, item T) Node) Node {
	nodes := make([]Node, 0, len(items))
	for i, it := range items {
		nodes = append(nodes, build(i, it))
	}
	return Frag(nodes...)
}
