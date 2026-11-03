status: todo

# Cancel and timeout

## Goal
A running request cannot be stopped and the 60 s timeout is hard-coded.

## Scope
- Send turns into Cancel while a request runs (button and Escape); the goroutine's context is cancelled and the response pane says "Cancelled".
- Per-request timeout and follow-redirects options in a small Settings pane of the request, defaults from app settings.
- Ignore a stale result if the tab was re-sent (sequence number).

## Acceptance
- Test with a slow `httptest` server: cancel returns promptly, a second send is not clobbered by the first's late result.
- `make check` passes.
