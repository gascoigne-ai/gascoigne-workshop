status: todo
kind: docs
commit: docs: fix drift in architecture, rules and decisions
scope: *.md,.harness/*
depends_on: 079-versioned-formats

# 080 Fix docs and reset the loop

From the audit: docs contradict each other or the code.

**Goal:** docs describe the code as it is, in the present tense.

Scope
1. `architecture.md`: Limits keeps real limits only (move ~60 feature bullets under feature headings); fix multipart (files are sent), IME, tab heights (42/40 vs 46/44), table header case; drop history wording ("used to", "supersedes"); fix the `## internal/ui` heading.
2. Remove or correct the five paths that do not exist: `proto/stream.go`, `proto/mcpschema.go`, `settings/sections.go`, `appearance/colors.go`, `cmd/gascoigne/pages.css`.
3. `rules.md`: `markup.Must` for `component.Must`; tokens live in `uitoolkit/core/theme.go`; add `environment` to the domains; adapter-to-adapter imports as `layers_test.go:89` allows.
4. `decisions.md`: fix 12 and 13 (prefs controller shape); record the exception: the name "Postman" is allowed where it names the import/export format.
5. Reset `tasks/loop.md` order and step 7.

Decision made by the user (2026-11-18): keep the Postman name, record the exception.

Acceptance: every path named in `.harness/*.md` exists (checked by a script run in this task); `make check` green.
