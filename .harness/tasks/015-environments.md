status: todo

# Environments and variables

## Goal
Variables live on the collection and there is no environment switcher.

## Scope
- Environments (Dev, Prod, ...) with their own variable tables, a switcher in the header, secret variables stored outside the collection files.
- `{{var}}` autocomplete popup in the URL, header and body editors; unresolved variables are highlighted as errors; hover shows the resolved value.
- Resolution order: request, folder, environment, collection.

## Acceptance
- Tests for resolution order, secrets never written to collection files, unresolved detection.
- `make check` passes.
