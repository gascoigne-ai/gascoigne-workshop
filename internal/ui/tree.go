package ui

import "github.com/gascoigne-ai/gascoigne-workshop/internal/model"

// treeRow is one visible line of the collections tree.
type treeRow struct {
	Depth  int
	Name   string
	Path   string // slash-joined names from the root; identifies the row
	Folder bool
	Open   bool
	Req    *model.Request // set for request rows
}

// treeRows flattens root into the rows that are visible given which
// folder paths are expanded. The root itself is row 0 at depth 0.
func treeRows(root model.Folder, expanded map[string]bool) []treeRow {
	rows := []treeRow{{Name: root.Name, Path: root.Name, Folder: true, Open: expanded[root.Name]}}
	if expanded[root.Name] {
		rows = appendFolder(rows, root, root.Name, 1, expanded)
	}
	return rows
}

func appendFolder(rows []treeRow, f model.Folder, path string, depth int, expanded map[string]bool) []treeRow {
	for _, sub := range f.Folders {
		p := path + "/" + sub.Name
		rows = append(rows, treeRow{Depth: depth, Name: sub.Name, Path: p, Folder: true, Open: expanded[p]})
		if expanded[p] {
			rows = appendFolder(rows, sub, p, depth+1, expanded)
		}
	}
	for i := range f.Requests {
		req := &f.Requests[i]
		rows = append(rows, treeRow{Depth: depth, Name: req.Name, Path: path + "/" + req.Name, Req: req})
	}
	return rows
}
