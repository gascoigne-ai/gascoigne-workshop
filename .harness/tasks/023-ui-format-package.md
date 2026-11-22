status: todo
kind: refactor
commit: refactor(ui): display formatting moves to ui/format
scope: internal/**, e2e/**, Makefile, .harness/**
not_before: 2026-01-01
depends_on: 022-storage-package

# kernel/textfmt and kernel/jsonview become ui/format; kernel is gone

Goal: only UI packages and `app` import `textfmt` and `jsonview` (respview, calltrace, settingsui). Merge them
into `internal/ui/format` (no toolkit imports). `internal/kernel` is empty afterwards: delete the directory (it
holds only moved files), the `kernel` zone and its rows. Follow the move recipe in `.harness/reorg.md`.

## Model
Model: unchanged.

## Acceptance
- `internal/kernel` no longer exists; `internal/CLAUDE.md` names the zones that exist.
- Common acceptance of `.harness/reorg.md`.
