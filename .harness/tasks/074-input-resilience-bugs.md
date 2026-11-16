# 074 Input resilience: remaining bugs

From the input audit of 2026-11-16 (static read of toolkit, request, collection, response, workspace and settings code). Items 1-15 of the audit are fixed and committed, one commit each, test first: miniyaml stray closer, tree keys with no rows, `Field` cursor after `OnChange`, `TextMenu` nil clipboard, jsonview preview runes, store "Environments" folder, `.secrets.json` mode, settings backup, remembered unreadable collections, session `Close` deadlocks, MCP stdio kill, cURL method quoting, `Expand` size limit, jsonview depth limit, ws ping between fragments, closing a tab stops its request and session.

**Goal:** no typed or imported input panics, hangs or silently loses data in the code paths below.

**Objectives**
1. A panic in a background goroutine becomes an error shown to the user, not a dead app.
2. Numeric input (settings, markup attributes, environment) cannot produce NaN, Inf, overflow or division by zero.
3. Importers and parsers keep a whole import alive when one item is odd, and say what they skipped.

Done since (one commit each, test first): `recover` in `bus.Run` plus connect and desktop dialog through it (the font scan in `settingsui` keeps its own goroutine: that package cannot import another domain's controller); slider range normalized; markup NaN/Inf refused; `Input.Scaled` and `GDK_SCALE` guards; timeout capped at 24 h; `<include>` depth limit; cell edit past the end; `assert` header capture; cURL clustered and attached flags; MCP waiter leak; Postman numeric values and string request; OpenAPI odd paths and operations (`Result.Skipped`, toast); gRPC reflection file cap; `.proto` imports cannot leave the import paths; `miniyaml` depth limit. Checked and left alone: the schema blow-up (width cubed, pinned by a test), MCP string ids (the session only sends integers), the `split` zero-size mismatch (harmless, only divides a ratio).

Scope (one commit per item, test first, regression comment above each test)
1. Out-of-range settings: `settings.Decode` does not call `Validate`. Decide with the user what a loaded out-of-range value does (clamp, or default with a status line), then implement. Ask first.
2. Postman import reports what it skips (unsupported auth types, graphql and file bodies) the way OpenAPI does (`Result.Skipped`, toast). OpenAPI `requestBody.$ref` gives no body silently: resolve it or report it.
3. Response view of a very large body (32 MiB): `RawLines`/`CodeLines`/`CompactLines` run on the UI thread. Cap or load in steps; the viewer says it is truncated. Ask the user for the limit and the wording first.
4. `assert` live evaluation on every keystroke (`assertpane.go:46`) re-parses the body per line: parse once per run. Test counts parses.
5. Docs: `architecture.md` Limits gets the caps now in code (variable expansion 16 MiB, JSON depth 256, YAML depth 100, include depth 32, timeout 24 h, reflection 512 files, MCP/WS behavior), in the same commit as item 3 or as its own `docs` commit.

Acceptance: every item has a test that failed before for the stated reason; `make check` green; this file lists nothing unfinished; the file is deleted in the commit of the last item.
