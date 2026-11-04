// Package uitoolkit is the GPU-drawn UI toolkit the workshop is built
// on. It knows nothing about requests, protocols or collections.
//
// Start at vml: it is where views are written (see its package doc and
// examples). Drop down to widget for the pieces it draws, use component to
// write a view as an HTML-style template, and theming for themes.
//
// The packages, each depending only on the ones listed before it:
//
//   - widget: theme files, colors, the drawing context (Ctx), text
//     fields, text areas, the code viewer and scrollbars.
//   - vml: a Go-native view DSL (Row, Col, Stack, Scroll, Split, Table,
//     Tabs...) that lays out, updates and draws a tree of elements.
//   - engine: frame pacing and input helpers for Ebitengine games.
//   - component: HTML-style templates that build the same vml trees.
//   - theming: turns appearance preferences into a widget.Theme.
//
// Applications import the toolkit packages directly; nothing under this
// directory may import the application's internal packages
// (see widget/layers_test.go).
package uitoolkit
