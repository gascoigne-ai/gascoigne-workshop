package port

// FolderEntry is one name in a folder.
type FolderEntry struct {
	Name string
	Dir  bool // a folder, or a link to one, so the browser can open it
}

// Folders reads the file system for the in-app file browser, the file
// chooser of a desktop without a dialog program.
type Folders interface {
	// List is what dir holds, in no particular order.
	List(dir string) ([]FolderEntry, error)
	// Home is the user's home folder: where the browser opens, and what a
	// typed ~ means.
	Home() (string, error)
}
