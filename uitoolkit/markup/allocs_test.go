//go:build !race

package markup

import (
	"fmt"
	"testing"
)

// allocsOf is the allocations of one build of src, built once before.
func allocsOf(t *testing.T, src string, own Scope, fallbacks ...Scope) float64 {
	t.Helper()
	template, err := Parse("t.vml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return testing.AllocsPerRun(20, func() {
		if _, err := template.Build(own, fallbacks...); err != nil {
			t.Fatal(err)
		}
	})
}

// A `for` bound the scope by copying it per item, so a template read from a
// big scope cost more for every item. The cost of a build must not follow the
// size of the scope.
func TestForCostDoesNotGrowWithScopeSize(t *testing.T) {
	items := make([]any, 50)
	for i := range items {
		items[i] = "item"
	}
	scopeOf := func(size int) Scope {
		scope := Scope{"xs": items}
		for i := 0; i < size; i++ {
			scope[fmt.Sprintf("name%d", i)] = i
		}
		return scope
	}
	const src = `<col><text for="x in xs">{{x}}</text></col>`
	small, large := allocsOf(t, src, scopeOf(5)), allocsOf(t, src, scopeOf(200))
	if small != large {
		t.Errorf("a build allocates %v times with 5 entries in scope and %v with 200", small, large)
	}
}
