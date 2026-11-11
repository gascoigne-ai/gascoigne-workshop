package widget

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

const (
	// ToastMax is how many toasts are shown; older ones are dropped beyond it.
	ToastMax = 5
	// ToastInfoFor is how long an info toast stays.
	ToastInfoFor = 3 * time.Second

	toastW      = 320
	toastH      = 36
	toastGap    = 6
	toastMargin = 8 // between the stack and the bottom inset
)

// Toast is a short message over the window.
type Toast struct {
	Text    string
	IsErr   bool
	Expires time.Time // zero: stays until clicked
}

// ToastStack is the toasts shown, oldest first.
type ToastStack []Toast

// Show adds a toast: an info one fades after ToastInfoFor, an error stays
// until it is dismissed. It reports whether anything was added; an error
// with the text of one already shown is not, since the same failure on
// every frame or send is one message.
func (stack *ToastStack) Show(text string, isErr bool, now time.Time) bool {
	toast := Toast{Text: text, IsErr: isErr}
	if isErr {
		for _, shown := range *stack {
			if shown.IsErr && shown.Text == text {
				return false
			}
		}
	} else {
		toast.Expires = now.Add(ToastInfoFor)
	}
	*stack = append(*stack, toast)
	if len(*stack) > ToastMax {
		*stack = (*stack)[len(*stack)-ToastMax:]
	}
	return true
}

// Prune drops the info toasts whose time is up and reports whether any went.
func (stack *ToastStack) Prune(now time.Time) bool {
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
func (stack *ToastStack) Dismiss(i int) { *stack = append((*stack)[:i], (*stack)[i+1:]...) }

// Toasts shows the stack in the bottom-right corner of a window of
// width by height, bottomInset above its lower edge (a status bar), oldest
// on top. A click dismisses the toast under it. Nil when there are none.
func Toasts(stack *ToastStack, width, height, bottomInset int) *El {
	if len(*stack) == 0 {
		return nil
	}
	total := len(*stack)*toastH + (len(*stack)-1)*toastGap
	return Col(W(toastW), Gap(toastGap),
		FloatAt(width-core.Gutter-toastW, height-bottomInset-toastMargin-total),
		For(*stack, func(i int, toast Toast) Node {
			border, color := core.Border, core.Text
			if toast.IsErr {
				border, color = core.Error, core.Error
			}
			return Row(H(toastH), PadX(core.Gutter), Bg(core.Panel), Border(border), Shadow(), OnClick(func() { stack.Dismiss(i) }),
				Text(toast.Text, Ellipsis(), Color(color), Grow(1)),
			)
		}),
	)
}
