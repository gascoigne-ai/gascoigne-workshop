# 002 Templates load from embedded .vml files

Objective: a view lives in `<name>.vml` next to its Go file, embedded with `//go:embed`, instead of a Go string.

Scope
1. `markup.Load(fsys fs.FS, name string)` returns a `Template`; parse errors name the file and line.
2. Test: a `.vml` file builds a tree whose `Dump` equals the inline version (the header template as the case).
3. One existing inline template (sentlog row) moves to a file as the proof; its Go-side test stays unchanged.

Acceptance: stdlib only; cache and `Reference()` unchanged; error text test with file:line.
Validation: `make check`; `-tags release` too.
