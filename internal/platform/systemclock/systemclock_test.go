package systemclock

import (
	"testing"
	"time"
)

func TestNowIsCurrentTime(t *testing.T) {
	before := time.Now()
	got := Clock{}.Now()
	if got.Before(before) || time.Since(got) > time.Minute {
		t.Fatalf("Now = %v, want about %v", got, before)
	}
}
