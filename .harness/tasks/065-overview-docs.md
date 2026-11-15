# 065 Docs: the architecture, the project and a 3000 ft overview

Asked for by the user. Do this after 060, 061 and 064 so the text is not written twice.

**Goal:** orientation in five minutes for someone who has not seen the project.

**Objectives**
1. A single overview (`.harness/overview.md`) says what the product is, how the window is laid out, and what each top-level directory holds.
2. One diagram shows the layers: `uitoolkit` (core, view, widget, markup, theming), the domains with their roles, ports and adapters, and the composition root in `cmd/workshop`.
3. It follows one request from the URL bar through the protocol registry to the response viewer.
4. It says where to add a protocol, a widget, a setting, a view.
5. It covers the project's organization: harness files, task workflow, `make` targets.

Scope
1. Draft from the code, not from old docs.
2. The diagram (text, in the file).
3. Link it from CLAUDE.md and the head of architecture.md; remove what it now duplicates.

Acceptance: every path and command named exists (checked by a grep script); `make check` green.
