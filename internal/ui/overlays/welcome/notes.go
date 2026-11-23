package welcome

import "strings"

// welcomeSection is one block of the release notes: a heading and its points.
type welcomeSection struct {
	Title string
	Items []string
}

// welcomeNotes are the release notes of this version, shown by the welcome
// dialog; CHANGELOG.md holds the same list for the release page.
var welcomeNotes = []welcomeSection{
	{"Early access", []string{
		"0.1.0 is an alpha: ready to use every day, and still growing fast, so details will keep improving.",
		"Your collections are plain files: read them, diff them and keep them in git.",
		"Your feedback shapes what comes next. Tell us what you find.",
	}},
	{"Requests", []string{
		"REST, WebSocket, SSE, MCP (tools explorer, streamable HTTP and stdio) and gRPC (reflection or .proto files, unary calls and streams) share one window.",
		"Params, headers, bodies (JSON, text, form, multipart, file), auth and assertions.",
		"{{variables}} from the collection, its folders and environments.",
	}},
	{"Collections", []string{
		"A collection is a folder of JSON files; secrets stay in a separate file out of version control.",
		"Open several side by side; create, rename, save, reorder, close and delete them from the sidebar.",
		"The tree supports rename, duplicate, move and delete; drag a request or folder onto another collection to move it, hold Alt to copy it, or use Move to... and Copy to... in its menu.",
		"Tabs and tree roots show the collection's color.",
	}},
	{"Responses", []string{
		"Pretty, raw and compact views, with foldable and searchable JSON.",
		"History, a timeline, a call log and a log of the requests as sent.",
	}},
	{"Import and export", []string{
		"Import cURL commands, OpenAPI specs and Postman collections; export to Postman.",
		"Client configs (.workshop.yaml, .json, .toml) in release builds.",
	}},
	{"Make it yours", []string{
		"Dark and light themes, JSON theme files and a colors page.",
		"Settings in one settings.json, with a form and a raw editor; every shortcut can be rebound.",
		"Send with Ctrl+Enter, jump to the URL with Ctrl+L, close a collection with Ctrl+Shift+W.",
		"Native file dialogs on macOS, Windows and Linux (zenity, kdialog, yad, qarma or matedialog), else an in-app file browser.",
	}},
}

// wrapped is sections with every point broken into lines of at most cols
// characters: a Lines column draws what it is given and never wraps.
func wrapped(sections []welcomeSection, cols int) []welcomeSection {
	out := make([]welcomeSection, len(sections))
	for i, section := range sections {
		out[i].Title = section.Title
		for _, item := range section.Items {
			out[i].Items = append(out[i].Items, wrap(item, cols))
		}
	}
	return out
}

// wrap breaks text at spaces into lines of at most limit characters; a word
// longer than that keeps a line of its own.
func wrap(text string, limit int) string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= limit:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
