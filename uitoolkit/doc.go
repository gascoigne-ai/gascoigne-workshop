// Package uitoolkit is the GPU-drawn UI toolkit the workshop is built
// on. It knows nothing about requests, protocols or collections.
//
// The toolkit is three packages, each depending only on the ones above it:
//
//   - widget: theme files, colors, the drawing context (Ctx), text
//     fields, text areas, the code viewer and scrollbars.
//   - vml: a Go-native view DSL (Row, Col, Stack, Scroll, Split, Table,
//     Tabs...) that lays out, updates and draws a tree of elements.
//   - engine: frame pacing and input helpers for Ebitengine games.
//
// Applications import the toolkit packages directly; nothing under this
// directory may import the application's internal packages
// (see TestToolkitDoesNotImportTheApplication).
package uitoolkit
