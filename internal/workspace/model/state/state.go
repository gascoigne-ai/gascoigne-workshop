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

// Tab is one open request, found again by its collection and folder path.
type Tab struct {
	// Collection is the folder of the request's collection (see CollectionAt).
	Collection string `json:"collection,omitempty"`
	Path       string `json:"path"`                // folder names and request name joined by "/"
	Pane       int    `json:"pane,omitempty"`      // selected request pane tab
	PaneLabel  string `json:"paneLabel,omitempty"` // its label, which survives a pane added before it; Pane is for older files
	Resp       int    `json:"resp,omitempty"`      // selected response pane tab
}

// State is everything remembered.
type State struct {
	Window    Window  `json:"window"`
	SidebarW  float64 `json:"sidebarW,omitempty"`
	PaneSplit float64 `json:"paneSplit,omitempty"`
	// Expanded are the open folders, by path, of the collection that is
	// not saved, or of the first one in a file written when one collection
	// was open at a time; ExpandedIn are those of saved collections, by the
	// collection's folder.
	Expanded   []string             `json:"expanded,omitempty"`
	ExpandedIn map[string][]string  `json:"expandedIn,omitempty"`
	Tabs       []Tab                `json:"tabs,omitempty"`
	Active     int                  `json:"active,omitempty"`
	Columns    map[string][]float64 `json:"columns,omitempty"`   // table column weights by table name
	SettingsW  int                  `json:"settingsW,omitempty"` // the settings dialog size the user chose; 0 is the default
	SettingsH  int                  `json:"settingsH,omitempty"`
	// Trusted are the commands and files the user allowed a request to
	// use, as model.Access keys ("command:...", "file:/abs/path").
	Trusted []string `json:"trusted,omitempty"`
}

// CollectionAt is the index in dirs, the folders of the open collections
// in order ("" for one not saved), of the collection ref names; -1 when it
// is not open. An empty ref is the collection that is not saved, else the
// first: a file written when one collection was open at a time names none.
func CollectionAt(ref string, dirs []string) int {
	for i, dir := range dirs {
		if dir == ref {
			return i
		}
	}
	if ref == "" && len(dirs) > 0 {
		return 0
	}
	return -1
}
