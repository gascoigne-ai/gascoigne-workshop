status: todo
kind: chore
commit: test(app): golden scene for the welcome dialog
scope: internal/*,.harness/*
depends_on: 087-inject-global-state

# 089 Golden scene for the welcome dialog

`architecture.md:272` says the welcome dialog's golden scene is "not added yet".

Scope
1. Add the scene to the screenshot list.
2. Regenerate goldens the way CI does (container) and look at the PNG.
3. Update `architecture.md`.

Acceptance: golden test passes in CI form; `make check` green.
