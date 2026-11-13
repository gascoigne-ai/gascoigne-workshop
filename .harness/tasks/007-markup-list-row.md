# 007 `list-row` element

Objective: `widget.ListRow` (a row with the `row` control look, hover tint, and a selected look) is an element, so the font list, the MCP explorer list and the sidebar tree do not each copy its look into templates. Row 3 wrote the look out twice (`fontrow.vml`, `fontrow-selected.vml`) because a color token cannot be left blank inside one template.

Scope
1. Test first: `<list-row selected="...">` builds the same `Dump` as `widget.ListRow(...)` for selected and not selected, plus a click test through the scope handler.
2. The element, its attribute row in Reference, the package doc index line, architecture.md.
3. Move `fontrow.vml` and `fontrow-selected.vml` to it (pins and goldens unchanged) and delete the second file.

Acceptance: no template spells out the list-row look by hand; `make check` green; golden PNGs unchanged.
Validation: as the migration rows (pins unchanged, goldens identical).
