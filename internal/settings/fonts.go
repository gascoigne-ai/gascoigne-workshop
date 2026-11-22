package settings

// FontStore finds installed fonts and reads their files.
type FontStore interface {
	// Scan lists the installed families, sorted by name.
	Scan() []Family
	// Load reads the regular and bold files of family.
	Load(family Family) (regular, bold []byte, err error)
}
