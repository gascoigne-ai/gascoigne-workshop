# 070 gRPC `.proto` file import

Known limitation: gRPC has no `.proto` import; methods come from server reflection only.

**Goal:** gRPC can be used against servers without reflection.

**Objectives**
1. The user can point a gRPC request at `.proto` file(s) and get services, methods and message schemas.
2. Import paths and well-known imports (`google/protobuf/*`) resolve.
3. Parse errors are shown with file and line.
4. Calls work the same as with reflection (unary, streaming kinds already supported).
5. The source (proto files or reflection) is saved with the request or collection.

Scope
1. Decide parser (see below); record in `decisions.md`; if a new dep, it needs approval in `rules.md`.
2. Descriptor source behind the gRPC port: reflection and proto files are two entries of a table.
3. UI to add files and import paths; stored in the request model, which stays free of UI.
4. Fixtures under `testdata/` from real `.proto` files, not invented.
5. Tests, `make check`, docs.

Acceptance: a call against a non-reflection server succeeds from a `.proto` file; `make check` green.

**Decided (user, 2026-11-15)**
- Parser: `github.com/bufbuild/protocompile`. Scope item 1 adds it to the approved deps in `rules.md` (approved by the user on 2026-11-15) and records it in `decisions.md`.
- Proto files and import paths are stored per request.
- Paths are saved relative to the collection when inside it; absolute otherwise.
