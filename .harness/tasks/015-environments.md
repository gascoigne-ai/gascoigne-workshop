status: done (highlighting inside the header and body editors is not done)

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

## Progress
- Done: `model.Environment`, `MergeVars` (environment over collection), header switcher with new / rename / delete, remembered choice (`settings.environment`), Vars pane edits the active environment, secret variables (masked in the table, values in `.secrets.json`, `environments/*.env.json` on disk), unresolved `{{vars}}` in the URL shown as errors.
- Left: `{{var}}` autocomplete popup, hover showing the resolved value, folder-level variables (order request, folder, environment, collection; today environment over collection), highlighting in header and body editors.
- Done later: the URL field tooltip shows what the URL resolves to and which names are not defined.
- Done later: folder-level variables (`Folder.Vars`, saved in the folder's `folder.json`) with the full resolution order (`Collection.VarsFor`: collection, environment, folders outermost first, so request-side folders win over the environment); the folder context menu entry `Variables` points the Vars pane at the folder (the root: the collection), selecting an environment resets it.
- Left: the `{{var}}` autocomplete popup and highlighting inside the header and body editors.
- Done later: `Ctrl+Space` in the URL field opens a menu of the variables that fit at the cursor (a partly typed `{{na` filters) and writes `{{name}}`; typing-time popups were not done because a popup would take the keyboard from the field.
- Accepted: variable highlighting stays in the URL field (headers are table cells and the body editor styles JSON); the hover value is the URL tooltip.
