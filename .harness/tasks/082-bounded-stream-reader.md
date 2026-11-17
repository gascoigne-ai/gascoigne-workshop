# 082 Bounded event-stream reader and missing fuzz targets

From the audit: the SSE reader (`sse.go`) and MCP reader (`mcp.go`) grow without bound when `data:` lines never end with a blank line; `readSSE` ignores the scanner error. `settings.Parse`, `engine.ParseBinding` and `model.ParseBulk` take untrusted text and have no fuzz target.

Scope
1. Regression test: an endless `data:` stream is cut off at a limit and returns an error.
2. One shared reader with a size limit that reports the scanner error; SSE and MCP use it.
3. Fuzz targets for it, `settings.Parse`, `ParseBinding`, `ParseBulk`; add to `FUZZ_TARGETS`.
4. Limits entries in `architecture.md`.

Acceptance: `make fuzz` runs the new targets without a crasher; `make check` green.
