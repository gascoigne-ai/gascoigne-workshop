package ui

// Rect is an axis-aligned pixel rectangle.
type Rect struct{ X, Y, W, H int }

// Contains reports whether the point is inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Inset shrinks r by n on every side, never below zero size.
func (r Rect) Inset(n int) Rect {
	return Rect{r.X + n, r.Y + n, max(r.W-2*n, 0), max(r.H-2*n, 0)}
}

// SplitLeft cuts a w-wide strip off the left; w is clamped to r.
func (r Rect) SplitLeft(w int) (left, rest Rect) {
	w = min(max(w, 0), r.W)
	return Rect{r.X, r.Y, w, r.H}, Rect{r.X + w, r.Y, r.W - w, r.H}
}

// SplitRight cuts a w-wide strip off the right; w is clamped to r.
func (r Rect) SplitRight(w int) (rest, right Rect) {
	w = min(max(w, 0), r.W)
	return Rect{r.X, r.Y, r.W - w, r.H}, Rect{r.X + r.W - w, r.Y, w, r.H}
}

// SplitTop cuts an h-high strip off the top; h is clamped to r.
func (r Rect) SplitTop(h int) (top, rest Rect) {
	h = min(max(h, 0), r.H)
	return Rect{r.X, r.Y, r.W, h}, Rect{r.X, r.Y + h, r.W, r.H - h}
}

// SplitBottom cuts an h-high strip off the bottom; h is clamped to r.
func (r Rect) SplitBottom(h int) (rest, bottom Rect) {
	h = min(max(h, 0), r.H)
	return Rect{r.X, r.Y, r.W, r.H - h}, Rect{r.X, r.Y + r.H - h, r.W, h}
}

// Fixed sizes of the window chrome.
const (
	sidebarW   = 260
	headerH    = 44
	tabsH      = 40
	urlH       = 56
	statusH    = 28
	paneTabsH  = 40
	sidebarTop = 44
)

// Layout is where each region of the window sits.
type Layout struct {
	SidebarHead, Tree     Rect
	Header, Tabs, URL     Rect
	Request, Response     Rect
	Status                Rect
	SidebarDivider, Split Rect
}

// Compute places every region for a window of w by h pixels.
func Compute(w, h int) Layout {
	window := Rect{0, 0, w, h}
	body, status := window.SplitBottom(statusH)
	sidebar, main := body.SplitLeft(min(sidebarW, w/3))
	head, tree := sidebar.SplitTop(sidebarTop)
	header, rest := main.SplitTop(headerH)
	tabs, rest := rest.SplitTop(tabsH)
	url, panes := rest.SplitTop(urlH)
	request, response := panes.SplitLeft(panes.W / 2)
	return Layout{
		SidebarHead: head, Tree: tree,
		Header: header, Tabs: tabs, URL: url,
		Request: request, Response: response,
		Status:         status,
		SidebarDivider: Rect{sidebar.X + sidebar.W - 1, 0, 1, body.H},
		Split:          Rect{response.X, response.Y, 1, response.H},
	}
}
