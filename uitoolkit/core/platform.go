package core

import "strings"

// PrimaryDown reports whether the platform's shortcut modifier is held:
// Command (or Control) on macOS, Control elsewhere. The Windows key is
// never a shortcut modifier.
func PrimaryDown(goos string, ctrl, meta bool) bool {
	return ctrl || (goos == "darwin" && meta)
}

// PlatformKeyText writes the "Ctrl" of a key string the way the platform
// names its primary modifier, "Cmd" on macOS. Key strings are stored and
// compared with "Ctrl"; this is only for showing them.
func PlatformKeyText(goos, keys string) string {
	if goos != "darwin" {
		return keys
	}
	return strings.ReplaceAll(keys, "Ctrl", "Cmd")
}
