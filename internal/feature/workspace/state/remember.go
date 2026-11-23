package state

import (
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Key is the comparable form of s: equal states have equal keys. The
// autosave pause is measured on it.
func Key(s State) string {
	data, _ := json.Marshal(s) // a State holds only plain values: it always encodes
	return string(data)
}

// Remember keeps the open tabs that are in the tree, those with a path, and
// says where the active one went: its index among the kept tabs, the first
// when it is not kept. An active index past the tabs is returned as it is.
func Remember(open []Tab, active int) ([]Tab, int) {
	var kept []Tab
	keptAt := make([]int, len(open)) // open index -> kept index, -1 when dropped
	for i, tab := range open {
		keptAt[i] = -1
		if tab.Path == "" {
			continue
		}
		kept = append(kept, tab)
		keptAt[i] = len(kept) - 1
	}
	if active >= 0 && active < len(open) {
		active = max(keptAt[active], 0)
	}
	return kept, active
}

// Reopened is a remembered tab with the request that was found for it.
type Reopened struct {
	Tab Tab
	Req *request.Request
}

// Reopen is the remembered tabs whose request find still has, and the
// index among them of the active tab (0 when it is gone).
func (s State) Reopen(find func(Tab) *request.Request) ([]Reopened, int) {
	var reopened []Reopened
	active := 0
	for i, tab := range s.Tabs {
		req := find(tab)
		if req == nil {
			continue
		}
		if i == s.Active {
			active = len(reopened)
		}
		reopened = append(reopened, Reopened{Tab: tab, Req: req})
	}
	return reopened, active
}

// RememberExpanded records the open folders, by path, of the collection at
// dir: under its folder, or in Expanded for the one that is not saved
// (dir ""). No paths records nothing.
func (s *State) RememberExpanded(dir string, paths []string) {
	if len(paths) == 0 {
		return
	}
	if dir == "" {
		s.Expanded = append(s.Expanded, paths...)
		return
	}
	if s.ExpandedIn == nil {
		s.ExpandedIn = map[string][]string{}
	}
	s.ExpandedIn[dir] = paths
}

// ExpandedByCollection is the open folders by collection folder; the key ""
// is the collection that is not saved.
func (s State) ExpandedByCollection() map[string][]string {
	byDir := map[string][]string{}
	if len(s.Expanded) > 0 {
		byDir[""] = s.Expanded
	}
	for dir, paths := range s.ExpandedIn {
		if dir != "" {
			byDir[dir] = paths
		}
	}
	return byDir
}

// Sizes are the remembered sizes, in range.
type Sizes struct {
	SidebarW     float64
	SettingsW    int
	SettingsH    int
	PaneSplit    float64
	HasPaneSplit bool // false: the pane split was not remembered or is out of range
}

// Sizes is what a hand-edited or damaged file may hold, put in range:
// sizes are never negative and the pane split is a share strictly between
// 0 and 1.
func (s State) Sizes() Sizes {
	sizes := Sizes{SidebarW: max(s.SidebarW, 0), SettingsW: max(s.SettingsW, 0), SettingsH: max(s.SettingsH, 0)}
	if s.PaneSplit > 0 && s.PaneSplit < 1 {
		sizes.PaneSplit, sizes.HasPaneSplit = s.PaneSplit, true
	}
	return sizes
}
