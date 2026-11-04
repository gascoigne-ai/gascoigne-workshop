status: todo

# Requests: the rest of what a REST client needs

## Goal
Sending works for plain calls. Missing for real APIs: path variables, cookies, proxy and TLS control, files as bodies, and scripted checks.

## Scope
- Path variables written `:id` in the URL, with a table row each, next to `{{vars}}`.
- Cookie jar per workspace: sent automatically, viewer and clear in the UI, `Set-Cookie` shown in the response.
- Proxy (HTTP, SOCKS5) and TLS settings: skip verification switch, custom CA, client certificate and key; per request and as defaults in settings.
- Binary file body and multipart file fields with a file picker (needs the file-dialog decision in task 029).
- Pre-request scripts and response assertions: decide the language first (a tiny expression language over status, headers and JSON paths is enough to start; no embedded JS engine without asking).
- Timeline with DNS, connect, TLS, first byte and transfer from `net/http/httptrace`, shown in the Timeline pane.

## Acceptance
- Tests against `httptest` servers (cookies set and sent, proxy used, TLS with a test CA, multipart with a file, assertions pass and fail).
- `make check` passes; architecture.md updated.

## Progress
- Done: timeline from `httptrace` (`model.Timing`, the Timeline pane lists total, DNS, connect, TLS, first byte, transfer, size); a cookie jar shared by all REST requests until the app closes (tab menu `Clear cookies`); `Request.InsecureTLS` (Settings pane checkbox `Verify the server certificate`, saved with the request); an HTTP proxy from `settings.proxy` (no UI, file only), all tested against `httptest` servers.
- Left: path variables `:id`, cookie viewer, SOCKS5 and custom CA and client certificates, file bodies (needs the file dialog), scripts and assertions.
