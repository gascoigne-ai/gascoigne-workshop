status: todo
kind: refactor
commit: refactor(layout): controller packages sit directly under their domain
scope: internal/**,cmd/**,Makefile,.harness/**
not_before: 2026-01-01
depends_on: 151-exampleserver-and-envs

# Drop the controller/ level

Part of the package re-layout (domain first, two levels). Decided by the user: layout C; the import-layer check does NOT come back; `settings/model/appearance` loses its toolkit import by moving the theme types out. Out of scope: `uitoolkit` renames/splits and code changes that shrink `App`.
Target shape: `internal/<domain>/` holds the domain root package (types and ports, named after the domain, so no `model` or `port` packages and no import aliases); sub-packages sit directly under it; only `adapter/` and `ui/` are zone directories; the composition root is `internal/app`.
A refactor keeps tests green and changes none of them except import paths and renamed identifiers. `make check` is green after the task. Docs that name a moved path change in the same commit (`internal/CLAUDE.md`, `.harness/overview.md`, `.harness/architecture*`, Makefile `FUZZ_TARGETS` and `RELEASE_BASE`).

Per domain, one commit-sized step in this order: response, workspace, collection, settings, request. `request/controller/editor` becomes `request/editor`. Pure moves. Fix name stutter only where a package name repeats its domain (e.g. `collection/controller/collections`) and record each rename in the report.
