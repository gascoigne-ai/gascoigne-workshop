status: todo
kind: feature
commit: feat(limits): every input bound is named and written in architecture.md
scope: internal/*,.harness/architecture.md
depends_on: 101-stale-comments

# Name and document every input limit

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

Untrusted-input rule: a parser has a limit written into `architecture.md` Limits.
- Name the unnamed literals: OpenAPI `$ref` depth 6 (`openapi.go`), image size cap `64<<20` (`respview/image.go`).
- Add to Limits: WebSocket `DefaultMaxMessage` 16 MiB, `mcpMaxMessage` 16 MiB, the two above.
- The WebSocket handshake reads the response with `http.ReadResponse` and no bound: bound it (header size) and test with an oversized response.
- Postman, cURL and client-config input size: state the limit or write down that the caller bounds it.
Acceptance: test for the handshake bound written first and seen failing; `make check` green.

Model: unchanged
