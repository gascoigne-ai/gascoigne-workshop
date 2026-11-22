status: todo
kind: refactor
commit: refactor(layout): one root package per domain holds types and ports
scope: internal/**,cmd/**,Makefile,.harness/**
not_before: 2026-01-01
depends_on: 153-drop-model-level

# Merge model and port into the domain root

Part of the package re-layout (domain first, two levels). Decided by the user: layout C; the import-layer check does NOT come back; `settings/model/appearance` loses its toolkit import by moving the theme types out. Out of scope: `uitoolkit` renames/splits and code changes that shrink `App`.
Target shape: `internal/<domain>/` holds the domain root package (types and ports, named after the domain, so no `model` or `port` packages and no import aliases); sub-packages sit directly under it; only `adapter/` and `ui/` are zone directories; the composition root is `internal/app`.
A refactor keeps tests green and changes none of them except import paths and renamed identifiers. `make check` is green after the task. Docs that name a moved path change in the same commit (`internal/CLAUDE.md`, `.harness/overview.md`, `.harness/architecture*`, Makefile `FUZZ_TARGETS` and `RELEASE_BASE`).

Per domain, one step each: response, workspace, collection, settings, request (last, fan-in 24). Merge `<domain>/model/<x>` and `<domain>/port` into `internal/<domain>`; `model.Request` becomes `request.Request`; the `reqport`, `rport`, `sport`, `cport` aliases go. Check for import cycles per domain BEFORE merging; a cycle that cannot be resolved by moving one type is written into this file as blocked (rules.md, Tasks), not worked around. `settings/model/userdir` (its doc says it sits in model to dodge a layer rule) goes where its users need it without that dodge.
