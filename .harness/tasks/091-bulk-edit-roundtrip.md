status: todo
kind: bug
commit: fix(model): bulk edit round-trips a line that starts with CR and #
scope: internal/*,.harness/*

# 091 Bulk edit does not round-trip a `#` line after a carriage return

Found by the new `FuzzParseBulk` (task 082). The input `"\r#"` parses to an enabled row named `#`, which formats as `#: ` and reads back as a disabled row. The `#` check in `ParseBulk` (`internal/request/model/rows.go`) trims only spaces and tabs, while the name goes through `TrimSpace`.

Rated frequency 1 x severity 2 = 2, below the fix threshold; filed here instead. Task 082 shipped `FuzzParseBulk` with a weaker property (at most one row per line, no panic) so it would not fail on this input. This task restores the real one.

Model: unchanged

Scope
1. Regression test: `ParseBulk(FormatBulk(rows)) == rows` for rows from the input `"\r#"`; watch it fail.
2. Make the `#` check and the name trim agree.
3. Restore the round-trip property in `internal/request/model/rows_fuzz_test.go`: `if again := ParseBulk(FormatBulk(rows), nil); !reflect.DeepEqual(again, rows) { t.Fatalf(...) }`; run the fuzz target briefly.

Acceptance: the round-trip property holds under fuzzing; `make check` green.
