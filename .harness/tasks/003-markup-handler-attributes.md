# 003 Handler attributes

Objective: markup can bind every handler a Go view can.

Scope (one commit each, each with a test that fails first comparing the template `Dump` with the Go DSL, pattern `TestRemainingAttributesMatchTheGoOptions`)
1. `on-right-click`, `on-middle-click`, `on-click-at`, `on-drag-over`.
2. `on-select`, `on-dismiss`, `on-change`, `on-enter`, `on-slide`, `on-slide-end`, `on-tab`.
3. Each has its row in `markup.Reference()` and the package doc index (rule 10).

Acceptance: `TestReferenceListsEveryElementAndAttribute` green; the Go options without an attribute list shrinks to the ones that say why in their doc comment.
Validation: `make check`.
