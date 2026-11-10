status: todo

# Choice chips and toggles in view

## Goal
The pattern `PrimaryButton` when selected else `Button` appears at 7 sites.

## Scope
- [open] `view.Choice(label, selected bool, opts ...Option)` and `view.Choices(items []string, selected int, opts...)` with `OnSelect`; width and height stay options.
- [open] Replace sites: `settingsui/form.go:70`, `app/panes.go:360-370,511`, `app/auth.go:106-135`, `request/ui/mcpform:149-170`, `shortcutpage.go` button var.
- [open] Decide in the task if `Toggle` (Auto-reconnect, Wrap: `wsbar.go:30`, `panes.go:499`) is the same element; if the semantics differ, ask the user.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
