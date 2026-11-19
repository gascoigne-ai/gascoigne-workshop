status: todo
kind: refactor
commit: refactor(markup): split component.go by concern
scope: uitoolkit/markup/*
depends_on: 118-split-app-sources

# Split markup/component.go

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Pure moves of `component.go` (1512 lines) into `elements.go`, `params.go`, `attributes.go`, `parse.go`, `build.go`, `scope.go`, `reference.go`, `hint.go`; put declarations in reading order (`finiteFloat` between `realParam`'s comment and body, `intsAttr` and `enumAttr` after `Reference`). The XML line-number closure and the name, line, message error type that exist twice (here and in `widget/svg.go`) are not part of this task.
Acceptance: `git diff -M` shows moves only; `make check` green.
