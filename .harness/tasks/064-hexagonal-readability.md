# 064 Make the hexagonal layout easy to read

Asked for by the user: revisit the UX code and analyze how to make the hexagonal (ports and adapters) pattern easier to read and understand.

**Goal:** a newcomer finds where code lives, and why, from package names and one picture.

**Objectives**
1. A written analysis of the `internal/<domain>/<role>/<package>` scheme: what is hard to read (five domains times five roles, the composition root's exemptions, the app package named `ui`, decision 11's package map).
2. Ranked proposals, with the user choosing which to do.
3. The chosen ones are applied with the layer test still enforcing the rules.
4. A package map (domain by role table) is generated from the layer test's own rules, so the docs cannot drift.

Scope
1. Analysis in this file: tree of `internal/`, packages per domain and role, `layers_test.go` rules and exemptions, how a request flows through the layers.
2. (decided, user's choice from the proposals) Rename the package of `internal/workspace/ui/app` from `ui` to `app`, dropping the import alias; renames with gopls, one commit. Not chosen: doc.go per package, moving the composition root to `cmd/workshop`.
3. (decided) A `make` target and a test print the domain by role package map from `internal/layers_test.go`; `architecture.md` shows it (or links the target); one commit.

Acceptance: `internal/layers_test.go` still enforces the rules; the map matches the tree (a test fails when a package is missing from it); no `ui "…/app"` alias remains; `make check` green.
