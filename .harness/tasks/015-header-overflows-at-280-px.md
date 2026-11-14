# 015 The header overflows a 280 px window

Found in task 009. In a 280 px wide window the header does not fit even with no environment: in `internal/workspace/ui/app/testdata/env-none-narrow.dump` the settings button is at x=253..297, past the inner edge at 268 (and past the window). The title (118 px) does not shrink, and the chip has at most 82 px there (256 inner, minus title 118, spacer 12 and button 44) while "No environment" is 111 px. Task 009 capped the chip at 200 px, which keeps the button inside from about 400 px up; with a long environment name the button is at x=342..386 in `env-active-long-narrow.dump`.

Scope
1. Regression test first: in a 280 px window the header's settings button stays inside the window, with no environment and with a long one.
2. Fix, by one of the options below; the pinned header dumps (`env-*-narrow`, `window-*-narrow` where the header is that narrow) change in the same commit, golden scenes looked at.

Options (needs a decision before work starts):
1. Shrink the title: it gets `ellipsis` and yields room before the chip and the button do.
2. Mark pieces `Optional` (the chip, or the title) so they are dropped when there is no room; the environment switcher or the title disappears in narrow windows.
3. A minimum window width at which the header fits, so 280 px cannot happen.

Acceptance: `make check` green.
