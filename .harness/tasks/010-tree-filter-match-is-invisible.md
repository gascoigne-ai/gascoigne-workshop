# 010 The part of a tree row the filter matched is nearly invisible

Found while adding the `tree-filter` golden scene (row 8). `rowName` in `internal/workspace/ui/app/view.go` draws the matched part in `core.AccentText`, which is the text color for an accent fill (`theming.go` pairs it with `core.Accent`). On the sidebar's sunken background in the dark theme it is a dark red that can hardly be seen: in `cmd/workshop/testdata/golden/tree-filter.png` the "Echo" of "Echo" and "WebSocket Echo" is near the background color.

Scope
1. Regression test first: the matched part of a row name is drawn in a token meant for text on the sidebar's background (a token such as `core.Accent` or `core.AccentLine`; ask the user which).
2. Fix in the sidebar view; the pinned dumps and the `tree-filter` golden scene change in the same commit, and the PNG is looked at in both themes.

Acceptance: `make check` green.
