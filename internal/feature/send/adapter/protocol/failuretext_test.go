package protocol

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestFailureWordsStoppedRequestsPlainly(t *testing.T) {
	boom := errors.New("boom")
	if got := failureText(fmt.Errorf("wrapped: %w", context.Canceled)); got != request.ErrCancelled {
		t.Errorf("cancelled: %q", got)
	}
	if got := failureText(fmt.Errorf("wrapped: %w", context.DeadlineExceeded)); got != request.ErrTimedOut {
		t.Errorf("timed out: %q", got)
	}
	if got := failureText(boom); got != "boom" {
		t.Errorf("other: %q", got)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithTimeout(context.Background(), time.Nanosecond)
	defer stop()
	<-expired.Done()
	for name, tc := range map[string]struct {
		ctx  context.Context
		want string
	}{
		"cancelled context hides the transport error": {cancelled, request.ErrCancelled},
		"expired context hides the transport error":   {expired, request.ErrTimedOut},
		"live context keeps the transport error":      {context.Background(), "boom"},
	} {
		if got := stopReason(tc.ctx, boom); got != tc.want {
			t.Errorf("%s: %q, want %q", name, got, tc.want)
		}
	}
}
