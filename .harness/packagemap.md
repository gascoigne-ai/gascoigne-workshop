# Package map

Generated from the tree and `internal/layers_test.go` by `make packagemap`; a test fails when it is out of date. Do not edit by hand.

Each package is `internal/<domain>/<role>/<package>`. The composition root, domain `workspace`, may also import the other domains' `ui` and `controller` packages.

| domain | model | port | controller | adapter | ui |
|---|---|---|---|---|---|
| collection | clientconfig, collection, miniyaml, openapi, postman | (root) | collections, transfer, tree | clientfile, store | destpicker |
| environment |  |  | envs |  |  |
| request | (root), assert | (root) | editor, mcpexplorer | exampleserver, proto, proto/ws | calltrace, mcpform |
| response | jsonview, textfmt | (root) | viewer | history, preview | respview, sentlog |
| settings | appearance, fileversion, fontfamily, settings, sysinfo, userdir | (root) | prefs | displayscale, fonts, keyrepeat, settingsfile, systeminfo | colorpage, settingsui, shortcutpage, systempage |
| workspace | severity, state | (root) | autosave, bus, filebrowser, overlay, serial, tabs | dockicon, filedialog, folders, statefile, systemclock | app, appicon |

What a role may import, in its own domain; another domain only through its `model` or `port`:

- `model`: model
- `port`: model, port
- `controller`: model, port, controller
- `adapter`: model, port, controller, adapter
- `ui`: model, port, controller
