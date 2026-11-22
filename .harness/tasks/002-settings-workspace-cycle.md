status: todo
kind: refactor
commit: refactor(layout): break the settings and workspace domain cycle
scope: internal/settings/**, internal/workspace/**
not_before: 2026-01-01
depends_on: 001-collection-root-no-sibling-imports

# Break the settings and workspace domain cycle

Goal: `settings/prefs` imports `workspace`, and `workspace` (statefile adapter, `settingsui`
consumers) imports `settings` leaves (`fileversion`, `userdir`), so the two domains depend on each other.
Pick which side owns the shared type and let the other import it one way only. Update the dependency
diagram and Known layer gaps in `.harness/architecture.md`.

## Model
Name the shared type, its owner and its persistence; one owner, the other side only imports it.

## Acceptance
- A refactor: tests stay green, none changes.
- Domain-level imports between `settings` and `workspace` run in one direction only.
- The repo check command passes.
