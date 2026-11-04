package tmpl

import (
	"io/fs"
	"sync"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// Loader finds templates by name in a file system and keeps them parsed.
// Over a directory (os.DirFS) a file whose modification time changed is
// read again on the next Build, so a layout can be edited while the
// program runs; over an embedded file system templates are read once.
type Loader struct {
	fsys fs.FS

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	tpl *Template
	mod time.Time
}

// NewLoader reads templates from fsys.
func NewLoader(fsys fs.FS) *Loader { return &Loader{fsys: fsys, cache: map[string]cached{}} }

// Template returns the parsed template called name.
func (l *Loader) Template(name string) (*Template, error) {
	info, err := fs.Stat(l.fsys, name)
	if err != nil {
		return nil, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if c, ok := l.cache[name]; ok && c.mod.Equal(info.ModTime()) {
		return c.tpl, nil
	}
	src, err := fs.ReadFile(l.fsys, name)
	if err != nil {
		return nil, err
	}
	tpl, err := Parse(name, src)
	if err != nil {
		return nil, err
	}
	l.cache[name] = cached{tpl: tpl, mod: info.ModTime()}
	return tpl, nil
}

// Build makes the view of the template called name.
func (l *Loader) Build(name string, sc Scope) (vml.Node, error) {
	tpl, err := l.Template(name)
	if err != nil {
		return nil, err
	}
	return tpl.Build(sc)
}
