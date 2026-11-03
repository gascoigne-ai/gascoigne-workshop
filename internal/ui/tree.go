package ui

import "github.com/gascoigne-ai/gascoigne-workshop/internal/model"

// treeRow is one visible line of the collections tree. Exactly one of
// Dir and Req is set; Folder says which.
type treeRow struct {
	Depth  int
	Name   string
	Folder bool
	Open   bool
	Dir    *model.Folder  // set for folder rows, including the root
	Req    *model.Request // set for request rows
}

// node is what the row stands for: the folder or the request.
func (r treeRow) node() any {
	if r.Folder {
		return r.Dir
	}
	return r.Req
}

// treeRows flattens root into the rows that are visible given which
// folders are expanded. The root itself is row 0 at depth 0. Folders
// are listed before the requests of their parent.
func treeRows(root *model.Folder, expanded map[*model.Folder]bool) []treeRow {
	rows := []treeRow{{Name: root.Name, Folder: true, Open: expanded[root], Dir: root}}
	if expanded[root] {
		rows = appendFolder(rows, root, 1, expanded)
	}
	return rows
}

func appendFolder(rows []treeRow, f *model.Folder, depth int, expanded map[*model.Folder]bool) []treeRow {
	for _, sub := range f.Folders {
		rows = append(rows, treeRow{Depth: depth, Name: sub.Name, Folder: true, Open: expanded[sub], Dir: sub})
		if expanded[sub] {
			rows = appendFolder(rows, sub, depth+1, expanded)
		}
	}
	for _, req := range f.Requests {
		rows = append(rows, treeRow{Depth: depth, Name: req.Name, Req: req})
	}
	return rows
}
