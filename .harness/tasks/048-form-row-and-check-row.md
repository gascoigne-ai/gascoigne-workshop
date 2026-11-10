status: todo

# Form row and check row in view

## Goal
Labeled rows (`Row(H(RowH), Gap, Text(label, W(...)), control)`) and clickable checkbox+label rows repeat across forms.

## Scope
- [open] `view.FormRow(label string, labelWidth float64, control Item, opts...)`; replace `app/auth.go:125,140`, `app/panes.go:429-434`, `settingsui/form.go:231-241`, `mcpform.go:112`.
- [open] `view.CheckRow(label string, on bool, opts...)` with `OnClick`; replace `app/panes.go:436,440`, `app/auth.go:116`, `mcpform.go:121`.

## Acceptance
- Test written first and seen failing for the right reason, then passing; `make check` green; `make shot` PNG checked for UI changes; `architecture.md` and the `view` package doc index updated in the same commit; this task file deleted in the last item's commit.

## Notes
- Follow toolkit API rules 1-11 in `rules.md`; options named `OnX`, flags without arguments, doc comment starting with the name.
