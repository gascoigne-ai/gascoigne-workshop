# Release checklist

Run before a tag `vX.Y.Z`. Tick a line with the date and what was seen; a line that cannot be done says why and what unblocks it.

## 0.1.0 (2026-11-15)

- [x] `make check` green (gofmt, vet, `go test -race`, and the same with `-tags release`), 2026-11-15, macOS.
- [x] `make shot` looked at: default window, a gRPC request (dividers, Stream chip), the header with the gear icon.
- [x] README current: features, install, build, collections, client configs, shortcuts, known limitations.
- [x] `CHANGELOG.md` has the version.
- [x] `release.yml` has a dry run (`workflow_dispatch`, builds, no release) and the release job runs only for a tag.
- [ ] **Blocked: golden PNGs.** The header changed (gear icon) and the gRPC strip changed, so `cmd/workshop/testdata/golden/*.png` need regenerating under software GL, which needs Linux with `xvfb-run`. Unblock: `xvfb-run -a go test ./cmd/workshop -run TestGoldenScenes -update`, look at a few PNGs, commit. CI's `make check` fails on the golden scenes until then.
- [ ] **Blocked: run the dry run on GitHub.** Needs the branch pushed; the program never pushes. Unblock: push, Actions, release, Run workflow; download the three artifacts.
- [ ] **Blocked: run the program on Windows and macOS** (decision 12): window, HiDPI, clipboard, file dialogs, browser opener, fonts, shortcuts. Unblock: a machine or a CI runner with a display.
- [ ] Tag and push `v0.1.0` (the user's step).

## Every release

- [ ] `make check` green, on Linux under `xvfb-run -a` so the golden scenes run.
- [ ] `version` in `internal/workspace/ui/app/app.go` is the tag.
- [ ] `CHANGELOG.md` and the README's feature list match what the program does.
- [ ] The dry run shows the Explorer icon on the Windows `.exe` (embedded by go-winres in the workflow, not run locally).
- [ ] The dry run of `release.yml` produced the Linux tarball, the macOS zip and the Windows zip.
