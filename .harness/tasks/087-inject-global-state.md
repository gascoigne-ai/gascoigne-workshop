status: todo
kind: chore
commit: refactor: inject host OS and retry delay
scope: internal/*,uitoolkit/*,.harness/*
depends_on: 086-data-model-cleanup

# 087 Inject global state

`core.HostOS` (`platform.go:10`) and `retryDelay` (`wsui.go:39`) are package variables that tests overwrite.

Scope
1. Test first: two cases run in parallel with different values.
2. Pass both through constructors.

Acceptance: no test assigns a package variable for these; `make check` green.
