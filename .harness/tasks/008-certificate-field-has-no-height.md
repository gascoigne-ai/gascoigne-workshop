# 008 The certificate path field is 0 px high once a path is set

Found while pinning the settings form (row 4). Once a certificate path is set in Settings, Requests, its text field lays out with height 0 (`fieldB grow=1 ... rect=0,278,268,0` in `internal/settings/ui/settingsui/testdata/form-requests-edited-narrow.dump`). It sits in a row with `AlignItems(AlignCenter)` and no height of its own, so nobody can click or right-click into a set path.

Scope
1. Regression test first: typing into the filled field and right-clicking it must reach the field (fails today); a layout test that its rect has a non-zero height.
2. Fix the view (the template that replaced `form.go`'s control); the pinned dumps that carry the 0 px field change in the same commit and the golden scenes are checked.

Acceptance: the field has the height of the other text fields; `make check` green.
