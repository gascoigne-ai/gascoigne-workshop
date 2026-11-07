package ui

import (
	"context"
	"os"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/filedialog"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// pickFile asks for a file: with the desktop's own dialog when there is
// one, else with a prompt for the path. It returns a function that closes
// the question without an answer (it kills the native dialog or closes the
// prompt), for a caller that has to take it back, as the settings dialog does
// when it is closed.
func (a *App) pickFile(title string, save bool, name string, done func(path string)) (cancel func()) {
	if !a.fileDialogAvailable() {
		a.prompt(title, name, "OK", done)
		asked := a.dialog
		return func() {
			if a.dialog == asked {
				a.closeDialog()
			}
		}
	}
	ctx, stop := context.WithCancel(context.Background())
	choose := a.chooseFile
	if choose == nil {
		choose = chooseWithSystemDialog
	}
	go func() {
		path, err := choose(ctx, title, save, name)
		switch {
		case err == filedialog.ErrCancelled || ctx.Err() != nil: // closed by the user, or taken back
		case err != nil:
			a.later <- func() { a.toast("File dialog failed: "+err.Error(), true) }
		default:
			a.later <- func() { done(path) }
		}
	}()
	return stop
}

func (a *App) fileDialogAvailable() bool {
	if a.hasFileDialog != nil {
		return a.hasFileDialog()
	}
	return filedialog.Available()
}

// chooseWithSystemDialog shows the desktop's own file dialog.
func chooseWithSystemDialog(ctx context.Context, title string, save bool, name string) (string, error) {
	if save {
		return filedialog.Save(ctx, title, name)
	}
	return filedialog.Open(ctx, title)
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
