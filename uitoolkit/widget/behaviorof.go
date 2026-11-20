package widget

import (
	"fmt"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// behaviorOf is the behavior of element as the kind T that option needs. An
// option put on another kind of element is a mistake in the caller's code,
// so it panics naming the option and the element's control instead of
// being ignored.
func behaviorOf[T any](element *El, option string) T {
	behavior, ok := element.Behavior().(T)
	if !ok {
		panic(fmt.Sprintf("%s: not an option of this element (control %q, behavior %T)", option, element.ControlName(), element.Behavior()))
	}
	return behavior
}
