package view

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// OnClick runs callback when the element is clicked. It sets the same
// handler as OnClickAt: the later of the two options wins.
func OnClick(callback func()) Option {
	return func(element *El) { element.onClick = func(int, int) { callback() } }
}

// OnClickAt is OnClick with the pointer position, for what opens at the
// click (a menu under the cursor); the app does not read the pointer itself.
// In a template on-click-at names a func(x, y int) of the scope. Like
// OnClick it sets the element's one click handler: the later option wins.
func OnClickAt(callback func(posX, posY int)) Option {
	return func(element *El) { element.onClick = callback }
}

// OnMiddleClick runs callback when the element is middle-clicked.
func OnMiddleClick(callback func()) Option { return func(element *El) { element.onMiddle = callback } }

// OnDoubleClick runs callback on the second click of a double click on the
// element, after its OnClick, which runs on both clicks: a list row
// selects on a click and opens on a double click.
func OnDoubleClick(callback func()) Option { return func(element *El) { element.onDouble = callback } }

// OnDragOver runs callback every frame the left or middle button is held with the pointer
// over the element, for drag and drop: the element under the dragged
// item learns the item is over it.
func OnDragOver(callback func()) Option { return func(element *El) { element.onDragOver = callback } }

// FloatTR takes the element out of the layout of its Stack and pins it
// top and right, this many pixels from those edges.
func FloatTR(top, right int) Option {
	return func(element *El) { element.fl.on, element.fl.top, element.fl.right = true, top, right }
}

// Modal makes a floating element swallow every click; a click outside it
// runs the OnDismiss handler.
func Modal() Option { return func(element *El) { element.modal = true } }

// OnDismiss runs when a click lands outside a Modal element.
func OnDismiss(callback func()) Option { return func(element *El) { element.onDismiss = callback } }

// If is Vue's v-if: the node when cond holds, else nothing.
func If(cond bool, node Node) Node {
	if cond {
		return node
	}
	return nil
}

// When applies opt only if cond holds: the option form of If, so a
// condition is never a bool parameter of the option itself.
func When(cond bool, opt Option) Option {
	if cond {
		return opt
	}
	return func(*El) {}
}

// OnRightClick runs callback with the pointer position when the element is
// right-clicked. It sets the same handler as OnRightClickIn: the later of
// the two options wins.
func OnRightClick(callback func(posX, posY int)) Option {
	return func(element *El) { element.onRight = func(_ *core.Ctx, posX, posY int) { callback(posX, posY) } }
}

// DismissMargin makes a click within px outside a modal element's rect
// count as a click on it, so it does not dismiss it: a resizable dialog
// grabs its edge there.
func DismissMargin(px int) Option { return func(element *El) { element.dismissMargin = px } }

// OnChange runs after the element's value was edited: the text of a field
// or an area, or a checkbox that was clicked. It is an attribute of any
// element, read by the component through ChangeHandler.
func OnChange(callback func()) Option { return func(element *El) { element.onChange = callback } }

// KeepInView marks the child a scrolling strip must keep in view: whenever
// a different child is flagged than last time, an HScroll scrolls to show
// it once, so a selection made elsewhere is never hidden, while the
// user's own scrolling is left alone. It is an attribute of the child, set
// here and read by its parent through KeepsInView.
func KeepInView() Option { return func(element *El) { element.reveal = true } }

// SelectHandler is the function given with OnSelect, nil for none. Widgets
// built on view read it when the user picks something, because the option
// is applied after they build their children.
func (e *El) SelectHandler() func(index int) { return e.onSelect }

// DismissHandler is the function given with OnDismiss, nil for none.
func (e *El) DismissHandler() func() { return e.onDismiss }

// ChangeHandler is the function given with OnChange, nil for none.
func (e *El) ChangeHandler() func() { return e.onChange }

// OnRightClickIn runs callback when the element is right-clicked, with the context
// and the pointer position. It is OnRightClick for a handler that needs
// the context, such as one that opens a popup menu. The later of it and
// OnRightClick wins.
func OnRightClickIn(callback func(ctx *core.Ctx, posX, posY int)) Option {
	return func(element *El) { element.onRight = callback }
}

// OnSelect runs with the index of the item the user picked: the tab of a
// Tabs strip, the row of a Menu.
func OnSelect(callback func(i int)) Option { return func(element *El) { element.onSelect = callback } }
