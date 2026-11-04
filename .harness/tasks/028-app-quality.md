status: todo

# App quality

## Goal
Make the app dependable to ship.

## Scope
- Error states for every failure a user can see (network, parse, disk, clipboard): one wording rule, a toast or an in-pane message, never a silent failure; a review pass over `internal/ui`.
- Window: minimum size, app icon, title with a dirty marker for unsaved edits.
- Text input: IME and dead keys.
- Golden screenshot tests for both themes (`make shot` compare against stored PNGs, with a tolerance) at 100% and 150%, since pixels cannot be read without a running game loop: drive them through the existing `-screenshot` mode.
- CI workflow (`make check` plus the screenshot compare), release packaging and a desktop entry.
- Accessibility pass: contrast is tested; add focus order review and screen reader hints where the platform allows.

## Acceptance
- A checklist of failure paths each with a test; screenshots compared in CI; `make check` passes.

## Progress
- Done: the window title shows the collection and a dot while something is unsaved (`App.Title`); a minimum window size of 720x480.
- Left: the rest.
