package core

import (
	"testing"
	"time"
)

func TestSamplerCountsClicksByItsOwnClock(t *testing.T) {
	now := time.Unix(100, 0)
	s := &Sampler{Now: func() time.Time { return now }}
	if got := s.now(); !got.Equal(now) {
		t.Fatalf("now = %v, want the injected clock %v", got, now)
	}
}
