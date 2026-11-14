# 059 A System page in the settings: capabilities of the system and GPU rendering

Asked for by the user: a settings page about the capabilities of the system and the GPU rendering (and, from the question before it, whether macOS runs on Metal). Read-only; the user sees what the app runs on. Contents chosen without a question, since there is a conventional answer: **Rendering** (graphics library: Metal, OpenGL, DirectX; GPU image memory; vsync; actual FPS and TPS), **Display** (monitor name and size, device scale factor, window size, interface scale), **System** (OS and architecture, CPU count, Go and Ebitengine versions, cgo, memory in use), and a Copy button that puts the report on the clipboard as text.

Scope
1. Model and port: `internal/settings/model/sysinfo` (`Report`, `Section`, `Row`, `Text()` for copying) and the port `SystemInfo` (`Read() sysinfo.Report`), tested.
2. Adapter `internal/settings/adapter/systeminfo`: builds the report from the runtime, build info and Ebitengine (graphics, monitor, vsync, FPS), the Ebitengine part behind an injected source so tests need no display; tested with a fake source.
3. Page `internal/settings/ui/systempage` (template, pinned layout) and the entry "System" in the settings dialog's list, drawn by the application's `ExtraPage`; the app wires the adapter; Copy uses the clipboard.
4. Docs: `architecture.md`, a decision, and `style.md` if the page sets a pattern.

Acceptance: the page shows the report in the dialog, Copy puts it on the clipboard, `make check` green, `make shot -page system` looked at.
