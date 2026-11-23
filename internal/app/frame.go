package app

// frame is what Layout works out from the window size.
type frame struct {
	LW, LH      int     // logical size the UI lays out in
	SW, SH      int     // screen size in device pixels
	Scale       float64 // device pixels per logical pixel
	WindowScale float64 // window units per logical pixel, for remembered sizes
}

// frameSize turns the window size (in the units the window system uses)
// into a frame. reported is the scale Ebitengine knows and actual the
// display's real one: they differ when the desktop scales the display
// without telling the window (Xwayland), whose window is then in real
// pixels. ui is the interface scale on top.
func frameSize(width, height int, reported, actual, ui float64) frame {
	scale := actual * ui
	return frame{
		LW: int(float64(width)*reported/scale + 0.5), LH: int(float64(height)*reported/scale + 0.5),
		SW: int(float64(width)*reported + 0.5), SH: int(float64(height)*reported + 0.5),
		Scale: scale, WindowScale: scale / reported,
	}
}

// uiScale is the interface scale: the user's setting times the platform's
// base factor.
func (a *App) uiScale() float64 { return a.window.UIScale() * a.baseScale }

// displayScale is the display's real scale: what the window system
// reports, unless that is 1 and the desktop says otherwise (a compositor
// that scales Xwayland windows itself).
func (a *App) displayScale(reported float64) float64 {
	if reported > 1 || a.display == nil {
		return reported
	}
	if factor := a.display.Factor(); factor > 0 {
		return factor
	}
	return reported
}
