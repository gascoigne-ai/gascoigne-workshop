# 074 Input resilience: remaining bugs

From the input audit of 2026-11-16 (static read of toolkit, request, collection, response, workspace and settings code). Items 1-15 of the audit are fixed and committed, one commit each, test first: miniyaml stray closer, tree keys with no rows, `Field` cursor after `OnChange`, `TextMenu` nil clipboard, jsonview preview runes, store "Environments" folder, `.secrets.json` mode, settings backup, remembered unreadable collections, session `Close` deadlocks, MCP stdio kill, cURL method quoting, `Expand` size limit, jsonview depth limit, ws ping between fragments, closing a tab stops its request and session.

**Goal:** no typed or imported input panics, hangs or silently loses data in the code paths below.

**Objectives**
1. A panic in a background goroutine becomes an error shown to the user, not a dead app.
2. Numeric input (settings, markup attributes, environment) cannot produce NaN, Inf, overflow or division by zero.
3. Importers and parsers keep a whole import alive when one item is odd, and say what they skipped.

Done since (one commit each, test first): `recover` in `bus.Run` plus connect and desktop dialog through it (the font scan in `settingsui` keeps its own goroutine: that package cannot import another domain's controller); slider range normalized; markup NaN/Inf refused; `Input.Scaled` and `GDK_SCALE` guards; timeout capped at 24 h; `<include>` depth limit; cell edit past the end; `assert` header capture; cURL clustered and attached flags; MCP waiter leak; Postman numeric values and string request; OpenAPI odd paths and operations (`Result.Skipped`, toast); gRPC reflection file cap; `.proto` imports cannot leave the import paths; `miniyaml` depth limit. Checked and left alone: the schema blow-up (width cubed, pinned by a test), MCP string ids (the session only sends integers), the `split` zero-size mismatch (harmless, only divides a ratio).

Also done: out-of-range settings clamped and named on load (decided: clamp and say so); a huge response body shown as its first 2 MiB with a note (decided: raw view of the first 2 MiB); Postman import lists what it skips; OpenAPI `requestBody.$ref` resolved; limits written into `architecture.md` Limits.

Dropped by the bug rating rule (frequency x severity below 5, issue instead of code): `assert` live evaluation parses the body once per `json` line on every keystroke. Nobody has measured a slow case; revisit if a large body with many assertions feels slow.

Scope: every item is done. The user decides whether this file is deleted (rules: ask before deleting).

Acceptance: met; `make check` green.
