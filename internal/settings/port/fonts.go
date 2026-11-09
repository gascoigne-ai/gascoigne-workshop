// Package port holds what the settings domain needs from the outside: the
// installed fonts and the settings file.
package port

import "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fontfamily"

// FontStore finds installed fonts and reads their files.
type FontStore interface {
	// Scan lists the installed families, sorted by name.
	Scan() []fontfamily.Family
	// Load reads the regular and bold files of family.
	Load(family fontfamily.Family) (regular, bold []byte, err error)
}
