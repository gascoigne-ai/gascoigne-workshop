package core

import (
	"reflect"
	"testing"
)

// A direct assignment to Ctx.Sizes (the field is exported) must show in the
// scope the next read returns, without a call that tells the Ctx.
func TestSizesScopeFollowsAnAssignedSizes(t *testing.T) {
	ctx := &Ctx{Sizes: SizesFor(16, 14)}
	if got := ctx.SizesScope()["rowH"]; got != 28 {
		t.Fatalf("rowH = %v, want 28", got)
	}
	ctx.Sizes = SizesFor(20, 16)
	if got, want := ctx.SizesScope()["rowH"], ctx.Sizes.Row; got != want {
		t.Errorf("rowH after an assignment = %v, want %v", got, want)
	}
}

func TestSizesScopeHasTheNamesOfSizesScope(t *testing.T) {
	ctx := &Ctx{Sizes: SizesFor(16, 14)}
	if got, want := ctx.SizesScope(), ctx.Sizes.Scope(); !reflect.DeepEqual(got, want) {
		t.Errorf("SizesScope = %v, want %v", got, want)
	}
}

func TestZeroCtxHasASizesScope(t *testing.T) {
	var ctx Ctx
	if got := ctx.SizesScope(); len(got) == 0 {
		t.Errorf("SizesScope of a zero Ctx is empty")
	}
}

// The scope is one map while the sizes stay, so a frame does not build 26
// entries per component.
func TestSizesScopeIsOneMapWhileTheSizesStay(t *testing.T) {
	ctx := &Ctx{Sizes: SizesFor(16, 14)}
	first, second := ctx.SizesScope(), ctx.SizesScope()
	if reflect.ValueOf(first).Pointer() != reflect.ValueOf(second).Pointer() {
		t.Errorf("two reads gave two maps")
	}
}

// Sizes.Scope stays a fresh map: callers write their own entries into it.
func TestSizesScopeMethodGivesAFreshMapEachCall(t *testing.T) {
	sizes := SizesFor(16, 14)
	first, second := sizes.Scope(), sizes.Scope()
	first["extra"] = 1
	if _, shared := second["extra"]; shared {
		t.Errorf("a write into one Scope shows in another")
	}
}
