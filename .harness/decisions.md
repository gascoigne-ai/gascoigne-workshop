# Decisions

1. **Ebitengine with custom widgets** (user choice). Pure-Go GPU 2D with Wayland/X11 support and one static-ish binary. Cost: every widget, text input and scroll area is ours. First milestone keeps input single-line and the code viewer read-only.
2. **Protocol registry.** REST is real in milestone 1; WebSocket and MCP are registered stubs that return "not implemented", so the UI path is exercised end to end.
3. **Sibling bare repo** like the other gascoigne projects; shared harness via `.harness/common`.
