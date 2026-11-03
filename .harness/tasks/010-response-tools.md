status: todo

# Response tools

## Goal
The response view shows pretty JSON only and can copy whole lines.

## Scope
- Character-level selection and copy in `CodeView` (drag, double-click word, Shift+arrows when focused).
- Find in response (Ctrl+F): incremental search, highlights, next/previous, match count.
- View modes: Pretty, Raw, Preview (images; HTML as text); line wrap toggle; binary bodies shown as a size summary.
- Copy body, copy as cURL (from the request), save body to a file (file dialog via an external helper such as zenity or kdialog when present, otherwise a path prompt).

## Acceptance
- Tests for search matching and navigation, wrap geometry, cURL generation (quoting, headers, body).
- `make check` passes.
