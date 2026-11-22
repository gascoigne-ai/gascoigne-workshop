// Package desktop reaches the desktop the window runs on: the Dock icon, and
// later the display scale, the key repeat settings and the system file
// chooser. Each needs an OS call or an OS command that Ebitengine does not
// offer, and each differs per OS, so the files are split by build tag.
//
// It lives in the toolkit because every program with a window needs the same
// four things; a copy per app would drift. It imports only core and knows
// nothing of the app.
package desktop
