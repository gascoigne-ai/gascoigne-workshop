status: todo

# Create, rename, delete, duplicate, move

## Goal
The collection tree is read-only.

## Scope
- Context menu on tree rows: New request, New folder, Rename (inline), Duplicate, Delete (with confirmation), Copy URL.
- Drag and drop to move requests and folders; drop indicator; expand on hover.
- Sidebar filter box (name and URL) with match highlighting.
- Keyboard: arrows navigate the tree, Enter opens, F2 renames, Delete deletes.

## Acceptance
- Model-level tests for each operation (names stay unique among siblings, moving into own subtree is rejected, open tabs follow renames and deletes).
- `make check` passes.
