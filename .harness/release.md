# Release checklist

Run before a tag `vX.Y.Z`. Tick a line with the date and what was seen; a line that cannot be done says why and what unblocks it.

## 0.1.0 (2026-11-15)

- [x] `make check` green (gofmt, vet, `go test -race`, and the same with `-tags release`), 2026-11-15, macOS.
- [x] `make shot` looked at: default window, a gRPC request (dividers, Stream chip), the header with the gear icon.
- [x] README current: features, install, build, collections, client configs, shortcuts, known limitations.
- [x] `CHANGELOG.md` has the version.
- [x] `release.yml` has a dry run (`workflow_dispatch`, builds, no release) and the release job runs only for a tag.
- [x] Golden PNGs regenerated for the gear icon, the gRPC strip and the dialog style, 2026-11-15, in a container like CI; the PNGs looked at, the whole set green and identical over three runs.
- [x] Golden PNGs regenerated for the bold collection root and Ctrl+Shift+W on the Shortcuts page (every scene that shows the tree), and `tree-two-collections` and `tree-empty` added, 2026-11-15, in a container like CI; the new PNGs and the changed ones looked at, the whole set green and identical over three runs.
- [x] Golden `file-browser` added (the in-app file browser with its footer hint, path field masked), 2026-11-15, in a container like CI; the PNG looked at, no other PNG changed, the whole set green and identical over three runs.
- [ ] **Blocked: run the dry run on GitHub.** Needs the branch pushed; the program never pushes. Unblock: push, Actions, release, Run workflow; download the three artifacts.
- [ ] **Blocked: run the program on Windows and macOS** (decision 12): window, HiDPI, clipboard, file dialogs, browser opener, fonts, shortcuts. Unblock: a machine or a CI runner with a display.
- [ ] Tag and push `v0.1.0` (the user's step).

## Every release

- [ ] `make check` green, on Linux under `xvfb-run -a` so the golden scenes run.

Goldens are drawn under software GL, so they are regenerated on Linux like CI, not on macOS: an `ubuntu:24.04` image with the apt packages of `.github/workflows/check.yml`, `xvfb` and Go from `go.mod`, run `--platform linux/amd64` with the repo mounted, then `xvfb-run -a go test ./cmd/workshop -run TestGoldenScenes -update`.
Look at every changed PNG, then run the set three times without `-update`: all green.
- [ ] `version` in `internal/workspace/ui/app/app.go` is the tag.
- [ ] `CHANGELOG.md` and the README's feature list match what the program does.
- [ ] The dry run shows the Explorer icon on the Windows `.exe` (embedded by go-winres in the workflow, not run locally).
- [ ] The dry run of `release.yml` produced the Linux tarball, the macOS zip and the Windows zip.
