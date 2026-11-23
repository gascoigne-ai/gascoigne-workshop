// Package notice owns the status line under the window.
//
// Only the status line lives here for now. The toasts stay in the app until
// the overlays move (task 043): a controller may not import the toolkit, and
// the toast stack, its limit and its de-duplication are toolkit code, so
// copying them here would make two sources of truth. The toasts join this
// package when their view leaves the app.
package notice

// Notices holds what the user is told. The status line is one entry: the
// last text set; "" means none.
type Notices struct {
	status string
}

// SetStatus replaces the status line.
func (notices *Notices) SetStatus(text string) { notices.status = text }

// Status is the status line, "" when there is none.
func (notices *Notices) Status() string { return notices.status }
