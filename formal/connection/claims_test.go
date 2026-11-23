package connection_test

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
		{ID: "refused-event-changes-nothing", Kind: formal.KindClaim, Statement: "An event the connection refuses in its current state changes nothing"},
		{ID: "session-only-while-up", Kind: formal.KindClaim, Statement: "A connection holds a session exactly while it is open or closing"},
		{ID: "user-close-never-retries", Kind: formal.KindClaim, Statement: "After the user disconnects, the connection never reconnects until the user connects again"},
		{ID: "retry-only-when-asked", Kind: formal.KindClaim, Statement: "A failed or dropped connection retries only when the tab asks for a reconnect, and otherwise ends idle"},
		{ID: "stale-connect-not-current", Kind: formal.KindClaim, Statement: "A connect the user gave up, or a newer connect replaced, is never the current one, so its late answer is dropped, and an open session is always the latest connect's"},
		{ID: "connecting-leaves", Kind: formal.KindClaim, Statement: "A connection in progress leaves Connecting once its answer arrives, and always has its answer on the way"},
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
