# 087 Inject global state

`core.HostOS` (`platform.go:10`) and `retryDelay` (`wsui.go:39`) are package variables that tests overwrite.

Scope
1. Test first: two cases run in parallel with different values.
2. Pass both through constructors.

Acceptance: no test assigns a package variable for these; `make check` green.
