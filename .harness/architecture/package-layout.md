## Package layout

The whole tree, one row per domain and one column per role, is in `.harness/packagemap.md`; it is
generated from the tree and the layer test (`make packagemap`), and a test fails when it is stale.

`internal/` is grouped by domain, then by role (decision 11): `internal/<domain>/<role>/<package>`.
What a role may import is written once, in `rules.md` (Layout of `internal/`) and in the generated
`packagemap.md`; `internal/layers_test.go` enforces it, with a table that proves each kind of
violation is caught, and `exceptions` there is empty.

Go allows one package per directory and a method cannot be declared in another package, so
everything that reads and writes the one `App` struct (tabs, panes, sidebar, environments,
workspace) is one package, `workspace/ui/app`. It imports the other domains' views (decision 11) and
gets their adapters through ports in `Options`, which `cmd/workshop` fills; `exceptions` in the
layer test is empty. A feature that needs little of the `App` is its own package with a `Host`
interface the App implements, a package doc and tests with a fake host (`calltrace`, `mcpform`,
`sentlog`, `settingsui`, `colorpage`, `shortcutpage`, `systempage`, `destpicker`). Adapters of a
port are handed to those views through the host (`Host.SentStore`, `Host.FontStore`).
