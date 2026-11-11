// Package examples holds the toolkit's runnable examples. They use the
// exported API only, which is why they are a package of their own and
// not files beside the code they show: go test runs them and checks
// their output, so they cannot go stale.
//
// Example_view, Example_textField and Example_button show views and
// inputs, Example_markupView, Example_markupParse and
// Example_markupParseErrors show templates, Example_theming and
// Example_themeColor show themes. Package hello is a small program with
// a window: go run ./uitoolkit/examples/hello.
package examples
