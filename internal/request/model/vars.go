package model

import (
	"sort"
	"strings"
)

// MergeVars is the variables a request sees when an environment is
// active: the enabled ones of over win against base by name. Expand
// replaces in order, so the winners come first.
func MergeVars(base, over []KV) []KV {
	out := make([]KV, 0, len(base)+len(over))
	shadow := map[string]bool{}
	for _, v := range over {
		if v.Enabled {
			out = append(out, v)
			shadow[v.Name] = true
		}
	}
	for _, v := range base {
		if !shadow[v.Name] {
			out = append(out, v)
		}
	}
	return out
}

// VarNames lists the names of the {{variables}} used in s, in order of
// appearance and without repeats.
func VarNames(s string) []string {
	var names []string
	seen := map[string]bool{}
	for {
		start := strings.Index(s, "{{")
		if start < 0 {
			return names
		}
		end := strings.Index(s[start:], "}}")
		if end < 0 {
			return names
		}
		name := s[start+2 : start+end]
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
		s = s[start+end+2:]
	}
}

// Defined reports whether vars has an enabled variable called name.
func Defined(name string, vars []KV) bool {
	for _, v := range vars {
		if v.Enabled && v.Name == name {
			return true
		}
	}
	return false
}

// Unresolved lists the variables used in s that vars does not define.
func Unresolved(s string, vars []KV) []string {
	var out []string
	for _, n := range VarNames(s) {
		if !Defined(n, vars) {
			out = append(out, n)
		}
	}
	return out
}

// VarsFor are the variables req is sent with. From lowest to highest
// priority: the collection's, the active environment's (nil for none),
// the variables of the folders above the request, outermost first, so
// the nearest folder wins. The request itself has no variables of its
// own: its fields are where a value is written.
func (c *Collection) VarsFor(req *Request, env *Environment) []KV {
	vars := c.Vars
	if env != nil {
		vars = MergeVars(vars, env.Vars)
	}
	var chain []*Folder // from the request's folder up to, not including, the root
	var node any = req
	for {
		parent, ok := c.Folder.Parent(node)
		if !ok || parent == &c.Folder {
			break
		}
		chain = append(chain, parent)
		node = parent
	}
	for i := len(chain) - 1; i >= 0; i-- {
		vars = MergeVars(vars, chain[i].Vars)
	}
	return vars
}

// VarCompletion is what completing a {{variable}} at cursor (in runes) of
// text offers: the names of the enabled variables that start with what is
// typed after an open "{{", sorted, and where that partial name starts (in
// runes), or -1 when the cursor is not inside braces and every name fits.
func VarCompletion(text string, cursor int, vars []KV) (names []string, from int) {
	runes := []rune(text)
	cur := min(max(cursor, 0), len(runes))
	before := string(runes[:cur])
	open := strings.LastIndex(before, "{{")
	from = -1
	prefix := ""
	if open >= 0 && !strings.Contains(before[open:], "}}") {
		from = len([]rune(before[:open]))
		prefix = before[open+2:]
	}
	seen := map[string]bool{}
	for _, v := range vars {
		if v.Enabled && v.Name != "" && !seen[v.Name] && strings.HasPrefix(v.Name, prefix) {
			seen[v.Name] = true
			names = append(names, v.Name)
		}
	}
	sort.Strings(names)
	return names, from
}

// UpsertVar sets the variable called name to value: an existing row gets
// the value and is switched on, else a row is added. It returns the rows.
func UpsertVar(rows []KV, name, value string) []KV {
	for i := range rows {
		if rows[i].Name == name {
			rows[i].Value, rows[i].Enabled = value, true
			return rows
		}
	}
	return append(rows, KV{Name: name, Value: value, Enabled: true})
}
