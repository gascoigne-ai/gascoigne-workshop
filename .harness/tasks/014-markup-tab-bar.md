# 014: tab-bar element in markup

## Objective
`<tab-bar>` builds what `widget.TabBar` builds (and `ScrollTabs` where a state is given), so templates stop spelling it out and `App.pane` (view.go ~l.401) can move to markup.

## Why
Two written-out copies (explorerlist.vml, requesttabs.vml are strips of the same kind) and `App.pane` is the third need: loop.md's third-copy rule.

## Scope
1. Read `widget.TabBar` (rows.go) and `widget.ScrollTabs` (tabs.go) and the existing `tabs` element in markup (check `markup.Reference()` first: some of this may exist). Add the missing element(s)/attributes mirroring the Go constructors (rule 9/10: kebab-case, doc comment, test comparing template Dump/behavior to the Go call written first and seen failing, `Reference()` row, package doc index). A typed `[]widget.TabItem` and `*ScrollState` bind by scope name (decision 52).
2. Replace the hand-written copies that are the same shape as `widget.TabBar` (explorerlist.vml; requesttabs.vml only if it IS that shape: the agent noted its sizes differ (tabsH 42, bg=frame, control=tabstrip), so leave it if it is not). Pins/testdata/PNGs unchanged (zero diff).
3. `loop.md` gap list and `architecture.md` element list updated; delete this file in the last commit.

## Acceptance
`make check` green; item 1 tests fail before and pass after; item 2 commit has zero test/testdata/PNG diff. If an element cannot mirror the Go constructor without a new option, stop and report.
