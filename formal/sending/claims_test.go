package sending_test

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
		{ID: "one-send-per-owner", Kind: formal.KindClaim, Statement: "A request has at most one send in flight, and the answer of a send that was cancelled or replaced is never accepted"},
		{ID: "send-numbers-unique", Kind: formal.KindClaim, Statement: "Every send gets a number larger than any before it, so an answer is told apart by its number"},
		{ID: "cancel-runs-once", Kind: formal.KindClaim, Statement: "Cancelling a send runs its cancel func once, however often Cancel is called"},
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
