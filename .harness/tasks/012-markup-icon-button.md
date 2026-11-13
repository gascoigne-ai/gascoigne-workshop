# 012: icon-button element in markup

## Objective
`<icon-button icon="search" on-click="..." tooltip="..."/>` builds what `widget.IconButton` builds, so templates stop spelling out its row tree.

## Why
Three templates spell it out (file-chosen.vml, findbar.vml, responsepane.vml): loop.md's third-copy rule.

## Scope
1. Element `icon-button` with required attribute naming the icon (look at `widget.IconButton`'s signature and the icon constructors, rule 9/10: name kebab-case, doc comment, row in `markup.Reference()` and package doc, test comparing template `Dump` to the Go call, written first and seen failing). Options that IconButton takes (size, lighten, bg, tooltip, on-click) are existing attributes where possible; add only what is missing.
2. Replace the spelled-out copies in the three templates (7 uses). Their pin tests, testdata and golden PNGs stay unchanged; the comments saying "markup has no icon-button element" go.
3. `loop.md`: remove the icon-button gap note; `architecture.md` markup element list updated.

## Acceptance
`make check` green; item 1 test fails before and passes after; item 2 commit has zero test/testdata/PNG diff; delete this file in the last commit.
