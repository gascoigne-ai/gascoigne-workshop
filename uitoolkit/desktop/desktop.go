// Package desktop reaches the desktop the window runs on. This package sets
// the macOS Dock icon: the window icon of Ebitengine does not reach the Dock,
// which shows the generic executable icon for a program started outside an
// app bundle (go run, an editor's run button). On other systems SetDockIcon
// does nothing. The sub-packages ask the desktop for its display scale
// (displayscale), its key repeat timing (keyrepeat) a file name
// (filedialog), the system report (systeminfo) and the installed fonts (fonts); each differs per OS and needs an OS call or command that
// Ebitengine does not offer.
//
// It lives in the toolkit because every program with a window needs the same
// things; a copy per app would drift. It imports nothing of the toolkit or the
// app.
package desktop
