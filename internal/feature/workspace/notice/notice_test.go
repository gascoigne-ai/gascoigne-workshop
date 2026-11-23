package notice

import "testing"

func TestTheStatusLineHoldsTheLastTextSet(t *testing.T) {
	var notices Notices
	notices.SetStatus("a")
	notices.SetStatus("b")
	if got := notices.Status(); got != "b" {
		t.Fatalf("status %q, want b", got)
	}
}
