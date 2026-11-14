# 013 The help text of the request Tests pane is one clipped line

Found in row 22. On the request Tests pane (`assertions.vml`) the help text is a single line cut off at the pane edge (visible in `cmd/workshop/testdata/golden/request-tests.png`). It should wrap.

Scope
1. Regression test first: at a narrow pane width the help text lays out as several lines inside the pane (no line wider than the pane).
2. Fix (wrap the text, e.g. `lines`); the pinned dumps `assertions-*` and the `request-tests` golden change in the same commit, PNG looked at.

Acceptance: `make check` green.
