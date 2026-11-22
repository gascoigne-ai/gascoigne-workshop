status: todo
kind: feature
commit: feat(shortcutpage): shortcuts as a filterable keymap table
scope: internal/settings/**, uitoolkit/widget/**, cmd/workshop/**, internal/workspace/**, .harness/style.md, .harness/architecture/settings.md
not_before: 2026-01-01
depends_on: 146-settings-row-layout

# Keymap table

Goal: the shortcuts page is presented like Zed's Keymap Editor (reference image /tmp/claude-1000/-home-lukaszb-Development-gascoigne-gascoigne-workshop-main/9915e3f2-18df-4415-b424-d658430aedc4/images/2.png; read it):
a filter box on top ("Filter action names...") with the page actions beside it (open the raw JSON,
create or edit a binding); a table with a header row and columns Action, Keystrokes (as key chips) and
Source (Default or User; add Context or Arguments columns only if our bindings have such a thing),
zebra rows, truncation with an ellipsis, a selected row that shows an edit pencil; a row is editable by
click or Enter through our existing recording flow (refusal and warning rules unchanged). Columns
follow the table widget's resize rules.

## Model
Model: unchanged unless the designer finds that Source (Default or User) cannot be derived from the
binding data today; then say so with a `## Model` section (one source of truth: derived, not stored).

## Scope
1. Look at `shortcutpage`, the `widget` table and the key chip look in `style.md`; design the steps.
2. Tests first (dump and layout at wide and narrow widths, no clipping; filter narrows the rows by
   action name; Source column; recording, refusal and warning flows keep working; keyboard
   navigation; the existing shortcut tests stay green or change only for the new layout).
3. Implement; sizes from Ctx.Sizes, colors from tokens.
4. Goldens: regenerate only `settings-shortcuts*` scenes (container recipe in `.harness/release.md`);
   look at each image.
5. Docs: `style.md`, `architecture/settings.md`.

## Acceptance
- The shortcuts page is a filterable table; every shortcut behavior is as before.
- `make check` passes.
