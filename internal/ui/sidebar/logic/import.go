package logic

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/importers/openapi"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/importers/postman"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
)

// ImportKind is the format of the clipboard text an import reads; it
// indexes Importers.
type ImportKind int

// The formats a folder can import.
const (
	ImportCurl ImportKind = iota
	ImportOpenAPI
	ImportPostman
	importKindCount
)

// Importer is one row of the table of importers: how a clipboard text is
// read and how a refusal and the parts left out are worded.
type Importer struct {
	Refusal     string // starts the toast of a text that is not of this kind
	SkippedWhat string // what the parts left out could not be ("parts could not be read")
	Parse       func(text string) (Imported, error)
}

// Imported is what an importer read, ready to be put into a folder.
//
// Attach puts it into dir, which is part of the collection owner (nil for
// none), under a name free there, opens the folders it made in view and
// returns that name and the request to show, nil when there is none.
// Skipped names the parts the importer left out.
type Imported interface {
	Attach(view *Tree, dir *request.Folder, owner *request.Collection) (name string, show *request.Request)
	Skipped() []string
}

// ImportedRequest is one request. It brings no variables and skips nothing.
type ImportedRequest struct{ Request request.Request }

// ImportedFolder is a folder of requests with the collection variables it
// brings and the parts that were left out. Variables the owner already
// defines are never overwritten.
type ImportedFolder struct {
	Folder   *request.Folder
	Vars     []request.KV
	Problems []string
}

// Importers is the table of importers by kind: an array, since the kinds
// are a small fixed enum (a map would only add a missing-key case). The
// cURL row reads with send.ParseCurl and names the request with
// send.ImportedName, which the importers cannot import (they are siblings
// of send); a view's logic may import both.
var Importers = [importKindCount]Importer{
	ImportCurl: {Refusal: "Not a cURL command: ", Parse: func(text string) (Imported, error) {
		req, err := send.ParseCurl(text)
		if err != nil {
			return nil, err
		}
		req.Name = send.ImportedName(req)
		return ImportedRequest{Request: req}, nil
	}},
	ImportOpenAPI: {Refusal: "Not an OpenAPI document: ", SkippedWhat: "parts could not be read", Parse: func(text string) (Imported, error) {
		res, err := openapi.Parse([]byte(text))
		if err != nil {
			return nil, err
		}
		return ImportedFolder{Folder: res.Folder, Vars: res.Vars(), Problems: res.Problems()}, nil
	}},
	ImportPostman: {Refusal: "Not a Postman collection: ", SkippedWhat: "things could not be imported", Parse: func(text string) (Imported, error) {
		res, err := postman.Parse([]byte(text))
		if err != nil {
			return nil, err
		}
		return ImportedFolder{Folder: res.Folder, Vars: res.Vars, Problems: res.Skipped}, nil
	}},
}

// Skipped is nothing: a request is read whole.
func (imported ImportedRequest) Skipped() []string { return nil }

// Attach adds the request to dir and shows it.
func (imported ImportedRequest) Attach(view *Tree, dir *request.Folder, _ *request.Collection) (string, *request.Request) {
	view.Expand(dir)
	stored := dir.AddRequest(imported.Request)
	return stored.Name, stored
}

// Skipped are the parts left out.
func (imported ImportedFolder) Skipped() []string { return imported.Problems }

// Attach appends the folder to dir, adds the variables owner lacks and
// shows the first request of the folder.
func (imported ImportedFolder) Attach(view *Tree, dir *request.Folder, owner *request.Collection) (string, *request.Request) {
	view.Expand(dir)
	imported.Folder.Name = dir.UniqueName(imported.Folder.Name)
	dir.Folders = append(dir.Folders, imported.Folder)
	if owner != nil {
		for _, kv := range imported.Vars {
			if !request.Defined(kv.Name, owner.Vars) {
				owner.Vars = append(owner.Vars, kv)
			}
		}
	}
	view.ExpandAll(imported.Folder)
	return imported.Folder.Name, tree.FirstRequest(imported.Folder)
}
