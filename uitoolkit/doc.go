// Package uitoolkit is the GPU-drawn UI toolkit the workshop is built
// on. It knows nothing about requests, protocols or collections.
//
// Start at view: it is where views are written (see its package doc and
// examples). Drop down to core for the fundamentals it draws with, use markup to
// write a view as an HTML-style template, and theming for themes.
//
// The packages, each depending only on the ones listed before it:
//
//   - core: the fundamentals: the Theme type and color tokens, the drawing
//     context (Ctx), pointer and keyboard input, the clipboard.
//   - view: a Go-native view DSL (Row, Col, Stack, Scroll, Split,
//     Table...) that lays out, updates and draws a tree of elements, with
//     the inputs: TextField, TextArea and CodeView and their elements
//     Field, Area and Code.
//   - engine: frame pacing and input helpers for Ebitengine games.
//   - widget: ready-made widgets composed from view: buttons, rows, tabs,
//     menus, dialogs and toasts. It depends on core and view.
//   - markup: HTML-style templates that build the same view trees.
//   - theming: everything that makes a core.Theme: the JSON theme files, the
//     dark and light palettes, color math, and appearance preferences.
//
// Applications import the toolkit packages directly; nothing under this
// directory may import the application's internal packages
// (see core/layers_test.go).
package uitoolkit
