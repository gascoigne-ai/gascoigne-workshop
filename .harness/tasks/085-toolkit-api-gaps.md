# 085 Toolkit API gaps

From the audit against Toolkit API rules 2, 9, 10 and the theme-token rule.

Scope
1. `DismissMargin` (`overlay.go:58`): template attribute `dismiss-margin`, or a doc line saying why none.
2. `VRuleInset` (`layout.go:399`): template row.
3. `HRule`/`VRule`/`VRuleInset`: add to the `view` package doc index; `HRule()`/`VRule()` take `opts ...Option`.
4. Gap token; use it and `{{rowH}}` instead of `h="28"` (`view.go:510-521`, `fontlist.go:207`, `calltrace.go:218`) and the ~60 `gap="6"`/`gap="8"`. Update `style.md`.

Acceptance: each new option has doc, test, index line, template row; `make shot` looked at, goldens unchanged; `make check` green.
