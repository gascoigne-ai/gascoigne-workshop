status: todo
kind: chore
commit: refactor(app): file access goes through ports
scope: internal/*,.harness/*
depends_on: 080-docs-fix

# 081 File access through ports

From the audit: the App reads and writes files directly on the UI goroutine (`respfile.go`, `treeops.go`, `filebody.go`, `shot.go`).

**Goal:** the composition root does no file I/O itself.

Scope
1. Test first with fake ports: save body, Postman export, file-body check, screenshot write.
2. Add ports (model/port role) and adapters; calls run off the UI goroutine.
3. Update `architecture.md`.

Acceptance: `workspace/ui/app` does not import `os`; `make check` green.
