// Package uitoolkit is the GPU-drawn UI toolkit the workshop is built
// on. It knows nothing about requests, protocols or collections.
//
// Start at vml: it is where views are written (see its package doc and
// examples). Drop down to core for the fundamentals it draws with, use component to
// write a view as an HTML-style template, and theming for themes.
//
// The packages, each depending only on the ones listed before it:
//
//   - core: the fundamentals: theme files and tokens, colors, the drawing
//     context (Ctx), pointer and keyboard input, the clipboard.
//   - vml: a Go-native view DSL (Row, Col, Stack, Scroll, Split, Table,
//     Tabs...) that lays out, updates and draws a tree of elements, with
//     the inputs: TextField, TextArea and CodeView and their elements
//     Field, Area and Code.
//   - engine: frame pacing and input helpers for Ebitengine games.
//   - component: HTML-style templates that build the same vml trees.
//   - theming: turns appearance preferences into a core.Theme.
//
// Applications import the toolkit packages directly; nothing under this
// directory may import the application's internal packages
// (see core/layers_test.go).
package uitoolkit
