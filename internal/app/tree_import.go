package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/importers/openapi"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/importers/postman"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	request "github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"path/filepath"
)

// imported is what a clipboard importer read: a folder or a single request,
// the collection variables it brings and the parts it left out.
type imported struct {
	folder  *request.Folder
	request *request.Request
	vars    []request.KV
	skipped []string
}

// importKind is one row of the importers: how a clipboard text is read and
// how a refusal and the parts left out are worded.
type importKind struct {
	refusal     string // starts the toast of a text that is not of this kind
	skippedWhat string // what the parts left out could not be ("parts could not be read")
	parse       func(text string) (imported, error)
}

var (
	curlKind = importKind{"Not a cURL command: ", "", func(text string) (imported, error) {
		req, err := send.ParseCurl(text)
		if err != nil {
			return imported{}, err
		}
		req.Name = send.ImportedName(req)
		return imported{request: &req}, nil
	}}
	openAPIKind = importKind{"Not an OpenAPI document: ", "parts could not be read", func(text string) (imported, error) {
		res, err := openapi.Parse([]byte(text))
		if err != nil {
			return imported{}, err
		}
		baseURL := request.KV{Name: openapi.BaseURLVar, Value: res.BaseURL, Enabled: true}
		return imported{folder: res.Folder, vars: []request.KV{baseURL}, skipped: append(res.Skipped, res.Warnings...)}, nil
	}}
	postmanKind = importKind{"Not a Postman collection: ", "things could not be imported", func(text string) (imported, error) {
		res, err := postman.Parse([]byte(text))
		if err != nil {
			return imported{}, err
		}
		return imported{folder: res.Folder, vars: res.Vars, skipped: res.Skipped}, nil
	}}
)

// importFolder adds what the clipboard holds, read as kind, to dir and
// opens it; a text of another kind is explained in a toast.
func (a *App) importFolder(dir *request.Folder, kind importKind) {
	text, err := a.ctx.Clipboard.Read()
	if err != nil {
		a.toast("Cannot read the clipboard: "+err.Error(), severity.Error)
		return
	}
	got, err := kind.parse(text)
	if err != nil {
		a.toast(kind.refusal+err.Error(), severity.Error)
		return
	}
	name := a.attachImport(dir, got)
	a.status = "Imported " + name
	if len(got.skipped) > 0 {
		a.toast(fmt.Sprintf("%d %s, first: %s", len(got.skipped), kind.skippedWhat, got.skipped[0]), severity.Error)
	}
}

// attachImport puts got into dir under a name free there and returns it.
func (a *App) attachImport(dir *request.Folder, got imported) string {
	a.expanded[dir] = true
	if got.request != nil {
		got.request.Name = dir.UniqueName(got.request.Name)
		a.openRequest(dir.AddRequest(*got.request))
		return got.request.Name
	}
	got.folder.Name = dir.UniqueName(got.folder.Name)
	dir.Folders = append(dir.Folders, got.folder)
	if col := a.ownerOf(request.Node{Dir: dir}); col != nil {
		for _, v := range got.vars {
			if !request.Defined(v.Name, col.Vars) {
				col.Vars = append(col.Vars, v)
			}
		}
	}
	a.expandAll(got.folder)
	if first := tree.FirstRequest(got.folder); first != nil {
		a.openRequest(first)
	}
	return got.folder.Name
}

// exportPostman writes dir and everything below it as a Postman v2.1
// collection to a file the user picks. Requests of kinds Postman has no
// place for are left out and counted in the message.
func (a *App) exportPostman(dir *request.Folder) {
	a.commitEdit()
	var vars []request.KV
	if col := a.ownerOf(request.Node{Dir: dir}); col != nil {
		vars = col.Vars
	}
	data, err := postman.Export(dir, vars)
	if err != nil {
		a.toast("Export failed: "+err.Error(), severity.Error)
		return
	}
	skipped := postman.Skipped(dir)
	name := dir.Name
	if name == "" {
		name = "collection"
	}
	a.pickFile("Export as Postman collection", filedialog.SaveFile, name+".postman_collection.json", func(path string) {
		a.writeFile(path, data, func(err error) {
			if err != nil {
				a.toast("Export failed: "+err.Error(), severity.Error)
				return
			}
			msg := "Exported to " + filepath.Base(path)
			if skipped > 0 {
				msg += fmt.Sprintf(" (%d requests of other kinds left out)", skipped)
			}
			a.toast(msg, severity.Info)
		})
	})
}
