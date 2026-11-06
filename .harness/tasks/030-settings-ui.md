status: todo

# Settings view and shortcut editor

## Goal
Settings are easy to find, change and inspect, in the style of the settings screens of VS Code or Zed.

## Scope
- Rework the settings view to that style: a search field at the top that filters settings across all categories, a category list on the left, and the settings of the chosen category on the right. Each setting shows a title, a one-line description and its control. Changed values are marked and can be reset one by one.
- Replace the shortcut list with a real editor: a table of actions with their keys, filtered by a search field that matches the action name or the key. Each row has a record button that captures the next key combination, with clear and reset-to-default actions. Conflicts with another action and keys that type text are shown as a warning (the checks `bindingWarning` already does) and the user decides.
- Settings live as JSON in the user config directory. `settings.json` is already stored there, so make that file the single source of truth, with every setting and every shortcut override in it, and define and document its schema.
- A raw JSON mode: a switch between the form view and the `settings.json` text. The text view has autocomplete for keys and enum values, taken from the same schema as the form, plus inline errors for unknown keys and wrong types. Saving from either view writes the same file, and a change on disk shows up in both views.

## Acceptance
- Tests first for: the settings schema (one table listing each key, type, default, allowed values and description, used by both views), search filtering, shortcut recording and conflict detection, and JSON validation and completion.
- The form and the raw text view agree after every edit. A raw edit with a syntax error leaves the last good settings in effect and shows the error.
- `make check` passes; screenshot checked with `make shot`; `architecture.md` and `decisions.md` updated.

## Notes
- The schema table is the foundation: the form, the raw editor, validation and completion all read it, so there is no second list of settings to keep in step.
- Completion needs a small JSON-aware editor mode. Decide in `decisions.md` whether to extend `TextArea` or build a separate editor; no new dependency.
- Related: 028 has "font family setting"; it should be a row in this schema, not a separate mechanism.
