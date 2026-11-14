# 011 An unbound shortcut shows the "Plain key" warning

Found pinning the shortcut page (row 18). An action with no binding shows "Plain key: works only while no text field has focus": `Warning(Binding{})` calls `TypesText()` on key 0, which returns true. Visible in `cmd/workshop/testdata/golden/settings-shortcuts-edited.png` and `internal/settings/ui/shortcutpage/testdata/unbound-narrow.dump` (both pin the bug today).

Scope
1. Regression test first: an unbound action row has no warning note.
2. Fix (`Warning` returns none for an empty binding); the pinned dump and the golden PNG change in the same commit, PNG looked at.

Acceptance: `make check` green.
