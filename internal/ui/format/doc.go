// Package format holds the display formatting the views share, with no
// toolkit import: durations and byte sizes (Duration, Size), JSON text
// helpers for editors (CompactJSON, IndentJSON, JSONStatus), and a response
// body turned into display lines (Lines, Compact, Tokenize): pretty printed
// JSON split into colorable spans, or plain text lines when the body is not
// JSON. Each span names a theme style, never a color, so the package stays
// free of the toolkit.
//
// It lives under ui and not in a shared core because only views and the app
// use it; it began as two small leaf packages (textfmt, jsonview) merged here so
// that nothing outside the ui layer owns display code. Revisit if a feature
// needs these helpers: then they move to the base layer.
package format
