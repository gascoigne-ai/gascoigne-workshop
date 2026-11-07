package ui

import (
	"context"
	"os"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/filedialog"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// pickFile asks for a file: with the desktop's own dialog when there is
// one (it runs off the UI goroutine and the answer comes back through
// a.later), else by asking for the path as text.
func (a *App) pickFile(title string, save bool, name string, done func(path string)) {
	if !filedialog.Available() {
		a.prompt(title, name, "OK", done)
		return
	}
	go func() {
		var path string
		var err error
		if save {
			path, err = filedialog.Save(context.Background(), title, name)
		} else {
			path, err = filedialog.Open(context.Background(), title)
		}
		switch {
		case err == filedialog.ErrCancelled:
		case err != nil:
			a.later <- func() { a.toast("File dialog failed: "+err.Error(), true) }
		default:
			a.later <- func() { done(path) }
		}
	}()
}

// fileBody is the editor of a file body: the path, and a button to pick it.
func (a *App) fileBody(t *reqTab) Node {
	sync := func() { t.req.Body = t.file.Text }
	size := ""
	if info, err := os.Stat(t.file.Text); err == nil && !info.IsDir() {
		size = "found"
	} else if t.file.Text != "" {
		size = "not found"
	}
	return Col(Grow(1), Gap(8),
		Row(H(w.RowH), Gap(6),
			Field(&t.file, Grow(1), Placeholder("Path of the file to send"), OnChange(sync), a.textMenuOpt(&t.file, sync)),
			Button("Choose...", W(90), OnClick(func() {
				a.pickFile("Choose the file to send", false, "", func(path string) {
					t.file = NewTextField(path)
					sync()
				})
			})),
		),
		Text(size, Small(), Muted(), When(size == "not found", Color(w.Error))),
		Text("The file is sent as it is, with a content type from its extension.", Small(), Muted()),
	)
}
