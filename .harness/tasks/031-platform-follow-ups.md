status: todo

# Platform follow-ups

## Goal
Finish what task 028 could not do without Windows and macOS machines.

## Scope
- Run the app on real Windows and macOS machines and fix what differs: window and HiDPI, clipboard, file dialogs, the browser preview opener, Cmd shortcuts, font discovery folders, IME. Blocked: needs those systems; unblocked by a user run or a CI runner with a display.
- Give the Windows `.exe` an icon. Blocked: embedding one needs a `.syso` resource made by a tool that is not an approved dependency; unblocked by the user approving one (the window icon is already set at run time).
- Confirm the release jobs (`release.yml`: macOS app bundle, Windows zip) on a real tag; they have never run.

## Acceptance
- A note per platform of what was run and what was found; fixes each with a test where one is possible; `make check` passes.
