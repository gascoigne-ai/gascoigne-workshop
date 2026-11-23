package escape_test

import (
	"flag"
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"
)

var update = flag.Bool("update", false, "write formal.lock from the claims")

const lockPath = "formal.lock"

// claims says in domain words what the tests of this package establish. Changing a sentence is a
// recorded decision (`formal accept`), so a claim cannot drift quietly.
func claims() []formal.Claim {
	return []formal.Claim{
		{ID: "escape-closes-topmost", Kind: formal.KindClaim, Statement: "Escape closes the layer drawn on top, whatever combination of layers is open"},
		{ID: "escape-never-stuck", Kind: formal.KindClaim, Statement: "Pressing Escape enough times closes every open layer, one at a time"},
		{ID: "keyboard-held-by-overlays", Kind: formal.KindClaim, Statement: "Text under a menu, dialog, picker, file browser or call log ignores keys, and under nothing else"},
	}
}

func TestClaimsMatchTheirLock(t *testing.T) {
	graph, err := formal.NewClaimGraph(claims())
	if err != nil {
		t.Fatal(err)
	}
	store := formal.LockFile(lockPath)
	if *update {
		if err := store.Save(formal.NewLock(graph)); err != nil {
			t.Fatal(err)
		}
	}
	formal.CheckLock(t, graph, store)
}
