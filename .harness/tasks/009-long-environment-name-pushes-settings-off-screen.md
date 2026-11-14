# 009 A long environment name pushes the settings button off screen

Found while pinning the environment chip (row 6). In a 280 px wide window a long active environment name stretches the chip to x=478 and the settings button starts at x=490, outside the window (`internal/workspace/ui/app/testdata/env-active-long-narrow.dump`). The chip has no maximum width and does not shrink or cut its text.

Scope
1. Regression test first: in a narrow window the header's settings button stays inside the window with a long environment name; the chip text is cut with an ellipsis.
2. Fix in `envchip.vml` (a max width and `ellipsis`); the pinned dump changes in the same commit; golden scenes unchanged (the chip shows "No environment" there).

Acceptance: `make check` green.

## Blocked (2026-11-14): needs a decision

The task's approach cannot pass its own regression test at the pinned 280 px width without changing the goldens:

- At 280 px the header overflows even with no environment: in `env-none-narrow.dump` the settings button is at x=253..297, past the inner edge at 268. The title (118 px) does not shrink either.
- At 280 px the chip has at most 82 px (256 inner, minus title 118, spacer 12 and button 44), but the "No environment" chip is 111 px. Any `max-w` that leaves the goldens unchanged (111 or more) leaves the button outside at 280 px.
- `flow.layout` never shrinks a content-sized child; a `grow` chip would take the free space and widen "No environment" in the goldens.
- Tried `max-w="200"` with `ellipsis` on the text: the chip is cut to 200 px, the settings button moves from x=490 to x=342 (still outside 280), and the text keeps its 328 px rect because a content-sized text inside the chip is not narrowed; the ellipsis also needs `grow="1"` on the text.

Options, recommended first:
1. `max-w` (a value to pick, at least 111) plus `ellipsis` and `grow` on the text, with the regression test at a width where title, chip and button fit (about 400 px); the 280 px overflow of the whole header becomes its own task.
2. A toolkit change: flow layout gives a child marked to yield min(content, free space), so the chip shrinks to what is left. Needs a decision entry.
3. Mark the chip `Optional` so it is dropped when there is no room (the environment switcher disappears in narrow windows).

Unblocks when the user picks one (and, for 1, the max width).
