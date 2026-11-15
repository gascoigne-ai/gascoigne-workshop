# 068 Linux file dialog without zenity or kdialog

Known limitation: on Linux native dialogs need zenity or kdialog; without them the program asks for a path.

**Goal:** a Linux user gets a usable file picker without installing anything extra.

**Objectives**
1. Behavior when neither tool exists is defined and good enough (see decision).
2. The user is told which tool would give native dialogs.
3. Open and save work the same through every path (zenity, kdialog, fallback).

Scope
1. Decide the approach (see below), record it in `decisions.md`.
2. Implement behind the existing file-dialog port; the adapter is chosen from a table, no switch on tool.
3. Unit tests of tool detection and command lines without a display; UI fallback verified with `make shot`.
4. Docs: README install notes, architecture.md.

Acceptance: on a Linux machine without zenity and kdialog a file can be opened and saved without typing a raw path blindly; `make check` green.

**Decided (user, 2026-11-15)**
- In-app file browser dialog built from toolkit widgets, used when neither zenity nor kdialog exists. No D-Bus dependency.
- Still said to the user: which tool gives native dialogs (objective 2).
