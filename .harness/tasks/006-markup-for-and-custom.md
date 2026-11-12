# 006 `for` as an element, `custom` through slot

Objective: lists and `view.Custom` behaviors are reachable from a template without special cases.

Scope
1. Test first: a template with the `for` directive on a list of structs builds the same tree as `For`; decide whether scope values may be typed structs and record it.
2. `slot` takes a `Custom` node; a test shows a custom behavior inside a template keeps its input and layout.

Acceptance: no migration needs Go view construction only to reach `For` or `Custom`.
Validation: `make check`.
