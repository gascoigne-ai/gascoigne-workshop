# 012 Screenshot runs with -set or -record write the real settings.json

Found in row 18. `-record` and `-set` save through `prefs.Edit` / `saveSettings`. The golden harness passes `-settings ""` so tests are safe, but a manual `workshop -screenshot ... -record/-set` without `-settings ""` writes the user's real settings file: `main.go` redirects only `-state` for screenshots (~line 121).

Scope
1. Regression test first: a screenshot run with `-set` or `-record` and no `-settings` leaves the default settings path untouched (sandbox HOME/XDG).
2. Fix: screenshot mode uses in-memory settings unless `-settings` is given. Ask the user if a screenshot should ever write settings.

Acceptance: `make check` green; architecture.md screenshot-mode line updated.
