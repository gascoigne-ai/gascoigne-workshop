package app

// appName is the program's name in the window title.
const appName = "Gascoigne Workshop"

// Title is the window title: the focused collection and the program, with
// a dot in front while that collection has changes not yet saved. For a
// collection without a folder, that is any of its tabs with an edit.
func (a *App) Title() string {
	entry, ok := a.focus()
	if !ok {
		return appName
	}
	col := entry.Collection
	unsaved := entry.Dir != "" && a.saveOf(col).Pending(a.keys.Collection(col))
	for _, t := range a.openTabs() {
		if entry.Dir != "" || unsaved || !a.keys.TabDirty(t.Tab) {
			continue // the owner is looked up last: it walks the tree
		}
		owner, _ := a.entryFor(t.Req)
		unsaved = owner.Collection == col
	}
	title := appName
	if col.Name != "" {
		title = col.Name + " - " + appName
	}
	if unsaved {
		title = dirtyMark + " " + title
	}
	return title
}
