# 005 Widget elements

Objective: `field`, `area`, `checkbox`, `code`, `image` as elements.

Scope: the field handlers `on-enter` and `on-tab` and the slider's `on-slide` and `on-slide-end` (`widget.OnEnter` and the others only work on their own element, so they come with it; task 003 did the generic ones). One element per commit, test first against the Go constructor (`widget.Field`, `Area`, `Code`...), state objects come from the scope. Reference rows and doc index lines.

From task 004, which bound typed state by name (decision 52) but added only the attributes its containers use: `segments` and `text-menu` (field), `slider-mid` (slider) and `font-source` (a `*text.GoTextFaceSource`, the font list) come with the element that takes them. `resizable-dialog` is not an element yet: `ResizableDialog`'s four bounds would read as `min-w`/`max-w`, which already name the size options (rule 5), so its attribute names need a decision before migration row 17.

Acceptance and validation as task 004.
