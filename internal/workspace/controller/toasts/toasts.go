// Package toasts keeps the short messages shown over the window: how long
// each stays, how many are shown and that a repeated error is one message.
package toasts

import "time"

const (
	// Max is how many toasts are shown; older ones are dropped beyond it.
	Max = 5
	// InfoFor is how long an info toast stays.
	InfoFor = 3 * time.Second
)

// Toast is a short message over the window.
type Toast struct {
	Text    string
	IsErr   bool
	Expires time.Time // zero: stays until clicked
}

// Stack is the toasts shown, oldest first.
type Stack []Toast

// Show adds a toast: an info one fades after InfoFor, an error stays until
// it is dismissed. It reports whether anything was added; an error with the
// text of one already shown is not, since the same failure on every frame
// or send is one message.
func (stack *Stack) Show(text string, isErr bool, now time.Time) bool {
	toast := Toast{Text: text, IsErr: isErr}
	if isErr {
		for _, shown := range *stack {
			if shown.IsErr && shown.Text == text {
				return false
			}
		}
	} else {
		toast.Expires = now.Add(InfoFor)
	}
	*stack = append(*stack, toast)
	if len(*stack) > Max {
		*stack = (*stack)[len(*stack)-Max:]
	}
	return true
}

// Prune drops the info toasts whose time is up and reports whether any went.
func (stack *Stack) Prune(now time.Time) bool {
	kept := (*stack)[:0]
	for _, toast := range *stack {
		if toast.Expires.IsZero() || now.Before(toast.Expires) {
			kept = append(kept, toast)
		}
	}
	removed := len(kept) != len(*stack)
	*stack = kept
	return removed
}

// Dismiss removes toast i.
func (stack *Stack) Dismiss(i int) { *stack = append((*stack)[:i], (*stack)[i+1:]...) }
