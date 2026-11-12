# 001 Dump lists every handler

Objective: `view.Dump` shows which handlers an element has, so a template and its Go view can be compared on behavior, not only look. Today only `onSelect` and `onDismiss` appear.

Scope
1. Test (fails first): two elements that differ only in `OnRightClick` dump differently; same for each handler option of `view`.
2. Dump prints every handler present, by name, in a fixed order.

Acceptance: all `view` handler options appear in `Dump`; existing Dump-based tests (`statusbar_test.go`) stay green unchanged.
Validation: `make check`.
