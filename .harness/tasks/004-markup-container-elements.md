# 004 Container elements

Objective: `scroll`, `tabs`, `table`, `dialog`, `menu`, `toasts` as elements, with the options they take (`vertical`, `separator`, `segments`, `active-tab`, `keep-in-view`, `focus-scope`, `min-first`, `min-second`, `as-ratio`, `slider-mid`, `font-source`, `text-menu`).

Scope: one element per commit, test first against the Go constructor's `Dump`; table and tabs take their spec and state objects from the scope (typed values, not strings). Decide in the first commit how typed scope values are passed and record it in `decisions.md`.

Acceptance: a template for each builds the same `Dump` and rects as the Go view; Reference rows and doc index lines present.
Validation: `make check`.
