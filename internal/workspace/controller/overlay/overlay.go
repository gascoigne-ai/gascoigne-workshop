// Package overlay decides which open thing Escape closes. Everything that
// can be open sits in one ordered list, so the order is stated once and not
// as the branches of a switch.
package overlay

// Layer is something Escape can close.
type Layer int

// The layers, in the order Escape closes them: the one drawn on top first.
const (
	ContextMenu Layer = iota // a right-click or dropdown menu
	Dialog                   // a prompt, confirmation or form
	CallLog                  // the call trace window
	FontList                 // the font list inside the settings dialog
	Settings                 // the settings dialog
	CellEdit                 // a table cell being edited
	FindBar                  // the response find bar
	Request                  // a running request, cancelled last
	layers
)

// Open is the set of layers that are open.
type Open [layers]bool

// Top is the layer one Escape closes, and false when nothing is open.
func (open Open) Top() (Layer, bool) {
	for layer := Layer(0); layer < layers; layer++ {
		if open[layer] {
			return layer, true
		}
	}
	return 0, false
}
