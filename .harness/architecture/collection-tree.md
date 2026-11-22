## Collection tree

- `request.Folder` holds pointers (`[]*Folder`, `[]*Request`), so a request keeps its identity (open
  tabs point at it) however its siblings change. The tree rows (`collection.Rows`) key the
  expanded set by folder pointer, so a renamed or moved folder stays open.
- The tree operations are pure functions in `internal/request/tree.go`; a move or copy is
  `collection/transfer`, which validates before it changes anything.
