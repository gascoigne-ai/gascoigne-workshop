status: todo
kind: chore
commit: build: portable make install, CI vet, pinned Go
scope: Makefile,.github/*,mise.toml,.harness/*
depends_on: 083-errors-and-contexts

# 084 Makefile and CI gaps

From the audit: `make install` fails on macOS (`install -Dm755` is GNU only, `Makefile:85-88`); macOS and Windows CI jobs skip `vet -tags release`; `mise.toml` pins `go = "latest"` while go.mod says 1.26.10; go-winres is used in `release.yml` but not approved.

Scope
1. `make install` works on macOS/BSD, or is Linux-only with a clear guard.
2. `vet -tags release` on macOS and Windows in `check.yml`.
3. Pin Go in `mise.toml` to the go.mod version.
4. `rules.md`: add go-winres to approved deps (approved by the user 2026-11-18); fix decision 12 (the Windows exe has an icon).

Acceptance: `make install` runs here; CI config reviewed; `make check` green.
