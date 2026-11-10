package bus

import "context"

// Cmd is an effect: network, file or timer work that must not run on the UI
// goroutine. It gets everything it needs when it is built, never reads the
// App, and returns the message that reports its outcome, or nil for none.
type Cmd func(ctx context.Context) Msg

// Run starts cmd on its own goroutine with ctx and dispatches its result.
func (queue *Bus) Run(ctx context.Context, cmd Cmd) {
	go func() {
		if msg := cmd(ctx); msg != nil {
			queue.Dispatch(msg)
		}
	}()
}
