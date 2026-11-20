## Collection tree

`model.Folder` holds `[]*Folder` and `[]*Request`, so a request keeps its identity (open tabs point
at it) however siblings change. `internal/request/model/tree.go` has the operations, all pure and
tested: `UniqueName` ("x", "x 2", ... across folders and requests), `AddRequest` / `AddFolder`,
`Parent`, `Contains`, `Rename` (trimmed, non-empty, unique among siblings), `Duplicate` (deep copy
placed after the original as "<name> copy"; returns the copy as a `Node`, the zero `Node` for the root, none or a node of no tree), `Delete`, all taking a `Node`, `Attach` (puts a detached node, kept by
pointer, into a folder before a sibling, renamed on a clash), `Node.Clone` (deep copy of a request
or folder) and `UsedVars` (the `{{variables}}` of a request's URL, enabled rows, bodies and
credentials, first-use order); a move or copy is `controller/transfer`, which validates before it
changes anything. The tree view's `treeRows` works on pointers and the expanded set is keyed by
folder pointer, so renaming or moving a folder does not collapse it.
