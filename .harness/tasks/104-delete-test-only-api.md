status: todo
kind: chore
commit: chore: delete API that only its own tests use
scope: internal/*,uitoolkit/*,cmd/*,.harness/*
depends_on: 103-delete-dead-code

# Delete API used only by its own tests

Origin: the repo audit of 2026-11-19 (three read-only reviews against the harness rules). The user approved all phases and, with that, the deletions and the gate change named here. Behavior stays the same unless the task says otherwise; tests stay green and none is loosened. Update `architecture.md` in the same commit where it describes what changes.

The user approved deleting these when nothing but a test of the item itself uses them (a test that only exercises a deleted item goes with it; no other test is touched or loosened). Check each; keep it and say why if production code or a test of other behavior needs it.
`markup.View` with `viewMu`, `viewCache`, `maxViewCache`; `markup.Define`, `<include>` and `Reference` (decision 4 keeps `Load` and `MustLoad`); `newMultiline`; `codeGeometry.colAt`; `areaGeometry.colAt`; `wrapLayout.firstRow`; `viewChip`; `newTab`; `dirty`; `ws.Conn.SetReadDeadline`; `calltrace.Log.SetFilter`; `SettingsDialog.loadedFaces`; `settings.Categories`; `appearance.ThemeFor`; `port.Registry.Names`; `ws.Options.TLS` and `MaxMessage` if no caller sets them (see the HTTP task before removing `TLS`: it will use it).
Acceptance: `make check` green; decisions.md updated where a decision mentions a removed item.
