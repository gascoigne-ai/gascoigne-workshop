# 058 A stronger dialog title bar, documented as the visual style

Asked for by the user after task 057: the title bar should read as a window title, show that it drags, and be able to say more than a name; and the visual style of dialog titles is written down in the harness. Chosen with the user: accent / stronger bar, drag affordance, subtitle (an icon was chosen too, but the bar's order of title then items leaves no place before the title; recorded in the style file as not done), and a new `.harness/style.md`.

Scope
1. (done) Stronger bar: `DialogTitle` on the `Frame` color (the window chrome) with a `Border` line under it (the toolbar keeps the soft one), a hover tint (`HoverBg(Hover)`) so the bar answers the pointer; pinned dialog dumps updated and looked at.
2. (done) Drag affordance: the pointer over the free part of a drag handle (not over a control, not on the border) shows the move cursor (`CursorMove`); tested in `view`.
3. (done) Subtitle: `DialogSubtitle(text)` is a muted small ellipsized text that sits after the title; the settings dialog shows its page (or settings.json in the raw view) and the Log dialog its entry count.
4. `.harness/style.md`: the visual style of dialog titles (structure, tokens, sizes, drag and close behaviour, what goes in the title bar and what in the toolbar, subtitle rules), linked from `rules.md`, and the decision in `decisions.md`.

Acceptance: `make check` green; `make shot` of the settings, a confirm and the Log dialog looked at; `.harness/style.md` matches the code.
