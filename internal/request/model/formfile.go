package model

import "strings"

// FormFile reports whether a multipart form value names a file to send:
// "@path" does; "@" and "@@x" are text ("@@x" sends the text "@x").
func FormFile(value string) (path string, ok bool) {
	path, ok = strings.CutPrefix(value, "@")
	if !ok || path == "" || strings.HasPrefix(path, "@") {
		return "", false
	}
	return path, true
}
