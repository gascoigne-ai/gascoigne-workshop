# Decisions

1. **Ebitengine with custom widgets** (user choice). Pure-Go GPU 2D with Wayland/X11 support and one static-ish binary. Cost: every widget, text input and scroll area is ours. First milestone keeps input single-line and the code viewer read-only.
2. **Protocol registry.** REST is real in milestone 1; WebSocket and MCP are registered stubs that return "not implemented", so the UI path is exercised end to end.
3. **Sibling bare repo** like the other gascoigne projects; shared harness via `.harness/common`.
4. **Toolkit split: `widget` + `vml` + app.** The reusable GPU toolkit and the view DSL are public packages with no knowledge of requests or protocols, so they can move to their own repo later. The app only holds state, handlers and views.
5. **VML is a Go-native DSL, not a template language.** Typed builder functions with chained attributes and Vue-like `If`/`For`/`Field`. The Go compiler checks it, refactors keep working, and no parser, reflection or generator is needed. A text template syntax can be added later as another front end that emits the same nodes.
6. **Views are rebuilt every frame.** State stays in the app; there is no virtual DOM diffing or reactive graph. Widgets that need identity across frames (text field, code view) take a pointer to app-owned state.
