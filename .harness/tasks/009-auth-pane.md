status: todo

# Authentication

## Goal
The Auth pane only says "No authentication configured".

## Scope
- Types: None, Bearer token, Basic (user/password), API key (header or query). Table-driven: each type declares its fields and how it modifies the request.
- Secret fields are masked with a show toggle; values may use `{{vars}}`.
- Auth is applied at send time and does not appear in the Headers table.

## Acceptance
- proto/ui tests per auth type (header or query result); masking; `{{var}}` expansion.
- `make check` passes.
