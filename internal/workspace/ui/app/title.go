package ui

// appName is the program's name in the window title.
const appName = "Gascoigne Workshop"

// Title is the window title: the collection and the program, with a dot
// in front while something is not saved: an edited request in a tab, or a
// change to the collection that has not been written to its folder yet.
func (a *App) Title() string {
	unsaved := a.workspace != "" && a.colSave.Pending(a.collectionKey())
	for _, t := range a.openTabs() {
		unsaved = unsaved || (a.workspace == "" && dirty(t))
	}
	title := appName
	if a.col.Name != "" {
		title = a.col.Name + " - " + appName
	}
	if unsaved {
		title = dirtyMark + " " + title
	}
	return title
}
