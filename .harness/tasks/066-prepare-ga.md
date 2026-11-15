# 066 Prepare for GA

Asked for by the user: update the docs and change the settings icon to a gear. Last of the batch (060 to 066), since it depends on the docs being final.

**Goal:** the first public release is credible.

**Objectives**
1. The settings button shows a gear icon (the "flip" asked for: replace the current icon).
2. User docs exist: install, `.workshop.*` client configs, shortcuts, collections.
3. The gaps of decision 12 (Windows and macOS untested, `release.yml` never run) are closed or listed as known limitations.
4. A release checklist exists and has been run.

Scope
1. Gear icon in `uitoolkit/widget/icons`, used by the header; pins updated; golden scenes regenerated and looked at.
2. README and user docs.
3. Release checklist: `make check` including `-tags release`, a `release.yml` run, changelog.
4. Known limitations list.

Acceptance: every checklist line ticked, or written here as blocked with the reason and what would unblock it.
