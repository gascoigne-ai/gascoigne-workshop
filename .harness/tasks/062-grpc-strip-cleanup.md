# 062 gRPC request strip: remove the odd pattern

Asked for by the user with a screenshot: in the gRPC strip a vertical bar stands before the method (`grpc.health.v1.Health/Check`), left of it the protocol name `gRPC` and right of it `Stream`, then the URL field. The bar looks like a stray caret and the method has no visible boundary.

**Goal:** the gRPC strip reads like the REST strip: each part is clearly a control.

**Objectives**
1. No stray bar before the method.
2. A divider sits between every chip of the strip (protocol, method, Stream) and none after the last; the method text has padding and a placeholder ("Method...") when empty.
3. `Stream` is a toggle chip with a visible on and off state (streaming works already; decision 2's "unary only" is stale and is fixed in the docs).

Scope
1. (decided, user's choice: dividers, padding, Stream chip) Cause found: `urlbar.vml` draws one divider after the protocol chip only, the gRPC method text is flush against it, and `Stream` is a bare label with no divider. Fix in `internal/workspace/ui/app/urlbar.vml` and `view.go` (`streamColor`).
2. Fix it; pin the view with a `view.Dump` test at two sizes, written to fail first.
3. Add a golden scene of a gRPC request if none shows the strip.
4. `style.md` if the fix sets a pattern for strips.

Acceptance: `make shot` of a gRPC request looked at and consistent with the other protocols; `make check` green.
