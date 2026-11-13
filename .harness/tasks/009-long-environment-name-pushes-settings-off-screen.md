# 009 A long environment name pushes the settings button off screen

Found while pinning the environment chip (row 6). In a 280 px wide window a long active environment name stretches the chip to x=478 and the settings button starts at x=490, outside the window (`internal/workspace/ui/app/testdata/env-active-long-narrow.dump`). The chip has no maximum width and does not shrink or cut its text.

Scope
1. Regression test first: in a narrow window the header's settings button stays inside the window with a long environment name; the chip text is cut with an ellipsis.
2. Fix in `envchip.vml` (a max width and `ellipsis`); the pinned dump changes in the same commit; golden scenes unchanged (the chip shows "No environment" there).

Acceptance: `make check` green.
