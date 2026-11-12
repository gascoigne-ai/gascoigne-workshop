package markup

import (
	"fmt"
	"io/fs"
)

// Load parses the template in the file name of fsys, usually an embed.FS next
// to the Go file that builds the view. A syntax error names the file and the
// line.
func Load(fsys fs.FS, name string) (*Template, error) {
	source, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("load template %s: %w", name, err)
	}
	return Parse(name, source)
}

// MustLoad is Load for an embedded file: a missing or malformed one is a
// mistake found the first time the program starts, the way Must is.
func MustLoad(fsys fs.FS, name string) *Template {
	template, err := Load(fsys, name)
	if err != nil {
		panic(err)
	}
	return template
}
