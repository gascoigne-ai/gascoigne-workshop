# 062 gRPC request strip: remove the odd pattern

Asked for by the user with a screenshot: in the gRPC strip a vertical bar stands before the method (`grpc.health.v1.Health/Check`), left of it the protocol name `gRPC` and right of it `Stream`, then the URL field. The bar looks like a stray caret and the method has no visible boundary.

**Goal:** the gRPC strip reads like the REST strip: each part is clearly a control.

**Objectives**
1. No stray bar before the method.
2. The method has a visible field boundary (or the picker's look) and a placeholder when empty.
3. `Stream` is either a real control or not shown while gRPC is unary only (decision 13).

Scope
1. Find the cause in the URL bar and the gRPC method picker (`internal/workspace/ui/app`, `view.go` and its template); write it here.
2. Fix it; pin the view with a `view.Dump` test at two sizes, written to fail first.
3. Add a golden scene of a gRPC request if none shows the strip.
4. `style.md` if the fix sets a pattern for strips.

Acceptance: `make shot` of a gRPC request looked at and consistent with the other protocols; `make check` green.
