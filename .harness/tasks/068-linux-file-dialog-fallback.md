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
- Items 1 and 2 are done (f1c090c to a22f09e): table of dialog programs per system, in-app browser (`controller/filebrowser`, `app/filebrowser.go`), `-browse DIR` shot flag, decision 18.
- Decided 2026-11-15: the browser starts in the user's home folder (not the working directory) and does not remember the last folder. Detection is widened so the browser is rarely needed: the dialog programs tried are ordered by the desktop (`XDG_CURRENT_DESKTOP`, `XDG_SESSION_DESKTOP`, `DESKTOP_SESSION`) and cover the common ones on Linux (zenity, kdialog, yad, qarma, matedialog and the like), through the same table, no D-Bus. The hint that names the tool to install is a muted footer line in the browser (default taken, the user did not choose a place).
