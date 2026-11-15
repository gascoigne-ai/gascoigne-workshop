# 064 Make the hexagonal layout easy to read

Asked for by the user: revisit the UX code and analyze how to make the hexagonal (ports and adapters) pattern easier to read and understand.

**Goal:** a newcomer finds where code lives, and why, from package names and one picture.

**Objectives**
1. A written analysis of the `internal/<domain>/<role>/<package>` scheme: what is hard to read (five domains times five roles, the composition root's exemptions, the app package named `ui`, decision 20's package map).
2. Ranked proposals, with the user choosing which to do.
3. The chosen ones are applied with the layer test still enforcing the rules.
4. Every package says its domain and role in its doc comment.

Scope
1. Analysis in this file: tree of `internal/`, packages per domain and role, `layers_test.go` rules and exemptions, how a request flows through the layers.
2. Ask the user to choose from the proposals (recommendation first).
3. Apply each chosen change in its own commit (renames with gopls; layer test and docs in the same commit).
4. A `doc.go` or package comment per package: domain, role, what it may import.

Acceptance: `internal/layers_test.go` passes; a grep finds a role statement in every package; `make check` green.
