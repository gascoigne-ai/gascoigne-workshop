// Package state is what the app remembers between runs: window geometry,
// open tabs, pane sizes. Reading and writing the file is the statefile
// adapter's job.
package state

// Smallest window the app restores.
const (
	MinWindowW = 640
	MinWindowH = 400
)

// Window is the window's size and position in device-independent pixels.
// A zero size means nothing was remembered.
type Window struct {
	W int `json:"w"`
	H int `json:"h"`
	X int `json:"x"`
	Y int `json:"y"`
}

// Clamp keeps the window inside a monitor of mw by mh: not larger, not
// smaller than the minimum, and not positioned off screen. An unset
// window stays unset.
func (w Window) Clamp(mw, mh int) Window {
	if w.W == 0 || w.H == 0 {
		return Window{}
	}
	w.W = max(MinWindowW, min(w.W, mw))
	w.H = max(MinWindowH, min(w.H, mh))
	w.X = max(0, min(w.X, mw-w.W))
	w.Y = max(0, min(w.Y, mh-w.H))
	return w
}

// Tab is one open request, found again by its folder path.
type Tab struct {
	Path string `json:"path"`           // folder names and request name joined by "/"
	Pane int    `json:"pane,omitempty"` // selected request pane tab
	Resp int    `json:"resp,omitempty"` // selected response pane tab
}

// State is everything remembered.
type State struct {
	Window    Window               `json:"window"`
	SidebarW  float64              `json:"sidebarW,omitempty"`
	PaneSplit float64              `json:"paneSplit,omitempty"`
	Expanded  []string             `json:"expanded,omitempty"` // folder paths
	Tabs      []Tab                `json:"tabs,omitempty"`
	Active    int                  `json:"active,omitempty"`
	Columns   map[string][]float64 `json:"columns,omitempty"`   // table column weights by table name
	SettingsW int                  `json:"settingsW,omitempty"` // the settings dialog size the user chose; 0 is the default
	SettingsH int                  `json:"settingsH,omitempty"`
}
