# 074 Input resilience: remaining bugs

From the input audit of 2026-11-16 (static read of toolkit, request, collection, response, workspace and settings code). Items 1-15 of the audit are fixed and committed, one commit each, test first: miniyaml stray closer, tree keys with no rows, `Field` cursor after `OnChange`, `TextMenu` nil clipboard, jsonview preview runes, store "Environments" folder, `.secrets.json` mode, settings backup, remembered unreadable collections, session `Close` deadlocks, MCP stdio kill, cURL method quoting, `Expand` size limit, jsonview depth limit, ws ping between fragments, closing a tab stops its request and session.

**Goal:** no typed or imported input panics, hangs or silently loses data in the code paths below.

**Objectives**
1. A panic in a background goroutine becomes an error shown to the user, not a dead app.
2. Numeric input (settings, markup attributes, environment) cannot produce NaN, Inf, overflow or division by zero.
3. Importers and parsers keep a whole import alive when one item is odd, and say what they skipped.

Scope (one commit per item, test first, regression comment above each test)
1. `internal/workspace/ui/bus/cmd.go` (`bus.Run`), `asyncwrite.go:24`, `wsui.go:90`, `filebrowser.go:50`: `recover` in one place, turn the panic into an error event and a toast. Test: a job that panics reports an error and the bus keeps working.
2. Slider (`widget/slider.go`): `step<=0`, `min>=max`, `Scale=0` give no NaN thumb. `markup` `realParam`/`floatAttr`: reject NaN and Inf. `core.Input.Scaled`: scale 0 or negative. `displayscale.go` `parseEnvScale`: reject Inf and NaN (`GDK_SCALE=inf`).
3. `app.go:477` `timeoutFor`: a huge `timeoutSec` overflows `time.Duration` and expires at once; clamp. `settings.Decode` does not call `Validate`: decide in the item what a loaded out-of-range value does (clamp, or default with a status line) and ask the user if it changes what they see.
4. `markup` `<include>`: depth limit so a self-including template is an error, not a stack overflow.
5. `celledit.go:50-65` commit index after the table shrank. `assert.go:108-116` `set x = header X` with no headers must fail, not capture "". `widget/split.go` `Layout` and `Update` disagree at zero size (`max(total-1, 0)` vs `1`).
6. `curlparse.go`: unknown `--flag=value` becomes a positional argument; clustered short flags (`-sSL`, `-XPOST`, `-uuser:pw`) are ignored silently. Parse them or report them.
7. `mcp.go:235`: response ids matched only as int64; accept a string or `1.0` id. `mcp.go:307`: waiter leaks when `write` fails.
8. Postman import: numeric variable `value` and v2.0 string `request` fail the whole import; skipped features (unsupported auth, graphql and file bodies) are counted and reported. OpenAPI: `x-` extensions at the paths level, boolean schemas, `requestBody.$ref`, one malformed operation must not abort the import; self-referencing schema blow-up (`openapi.go:149-173`, 20^6 copies) gets a size cap.
9. `grpcreflect.go:124-138` dependency fetch loop and `grpcfiles.go:119-135` `.proto` import path (`../`): bound the loop; import outside the import path is refused (see task 076 for the trust side).
10. Response view of a very large body (32 MiB): `RawLines`/`CodeLines`/`CompactLines` run on the UI thread. Cap or load in steps; the viewer says it is truncated. Ask the user for the limit and the wording.
11. `miniyaml` very deep `[[[[` or `- - -` recursion: depth limit as an error with a line number. `assert` live evaluation on every keystroke (`assertpane.go:46`) re-parses the body per line: parse once per run.
12. Docs: `architecture.md` gets each limit and the `recover` rule in the same commit as the behavior.

Acceptance: every item has a test that failed before for the stated reason; `make check` green; this file lists nothing unfinished; the file is deleted in the commit of the last item.
