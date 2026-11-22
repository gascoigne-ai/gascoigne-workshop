package collection

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// Row is one visible line of the collections tree. Exactly one of
// Dir and Req is set; Folder says which.
type Row struct {
	Depth  int
	Name   string
	Folder bool
	Open   bool
	Dir    *request.Folder  // set for folder rows, including the root
	Req    *request.Request // set for request rows
}

// Node is what the row stands for: the folder or the request.
func (r Row) Node() request.Node {
	if r.Folder {
		return request.Node{Dir: r.Dir}
	}
	return request.Node{Req: r.Req}
}

// Rows flattens root into the rows that are visible given which
// folders are expanded. The root itself is row 0 at depth 0. Folders
// are listed before the requests of their parent.
func Rows(root *request.Folder, expanded map[*request.Folder]bool) []Row {
	rows := []Row{{Name: root.Name, Folder: true, Open: expanded[root], Dir: root}}
	if expanded[root] {
		rows = appendFolder(rows, root, 1, expanded)
	}
	return rows
}

func appendFolder(rows []Row, folder *request.Folder, depth int, expanded map[*request.Folder]bool) []Row {
	for _, sub := range folder.Folders {
		rows = append(rows, Row{Depth: depth, Name: sub.Name, Folder: true, Open: expanded[sub], Dir: sub})
		if expanded[sub] {
			rows = appendFolder(rows, sub, depth+1, expanded)
		}
	}
	for _, req := range folder.Requests {
		rows = append(rows, Row{Depth: depth, Name: req.Name, Req: req})
	}
	return rows
}

// FilteredRows is Rows narrowed to what matches filter: requests
// whose name or URL contains it (case-insensitive), the folders that
// hold one, and whole folders whose own name matches. Matching folders
// are shown open. The root is always there; an empty filter shows the
// tree as usual.
func FilteredRows(root *request.Folder, expanded map[*request.Folder]bool, filter string) []Row {
	if strings.TrimSpace(filter) == "" {
		return Rows(root, expanded)
	}
	needle := strings.ToLower(strings.TrimSpace(filter))
	rows := []Row{{Name: root.Name, Folder: true, Open: true, Dir: root}}
	return appendMatching(rows, root, 1, needle, strings.Contains(strings.ToLower(root.Name), needle))
}

func appendMatching(rows []Row, folder *request.Folder, depth int, needle string, all bool) []Row {
	for _, sub := range folder.Folders {
		subAll := all || strings.Contains(strings.ToLower(sub.Name), needle)
		if !subAll && !folderHasMatch(sub, needle) {
			continue
		}
		rows = append(rows, Row{Depth: depth, Name: sub.Name, Folder: true, Open: true, Dir: sub})
		rows = appendMatching(rows, sub, depth+1, needle, subAll)
	}
	for _, req := range folder.Requests {
		if all || requestMatches(req, needle) {
			rows = append(rows, Row{Depth: depth, Name: req.Name, Req: req})
		}
	}
	return rows
}

func requestMatches(call *request.Request, needle string) bool {
	return strings.Contains(strings.ToLower(call.Name), needle) || strings.Contains(strings.ToLower(call.URL), needle)
}

func folderHasMatch(folder *request.Folder, needle string) bool {
	for _, r := range folder.Requests {
		if requestMatches(r, needle) {
			return true
		}
	}
	for _, sub := range folder.Folders {
		if strings.Contains(strings.ToLower(sub.Name), needle) || folderHasMatch(sub, needle) {
			return true
		}
	}
	return false
}

// SplitMatch cuts text around the first case-insensitive occurrence of
// filter, so a row can color the part the filter matched. Without a
// match the whole text is returned as before.
func SplitMatch(text, filter string) (before, match, after string) {
	needle := strings.TrimSpace(filter)
	if needle == "" {
		return text, "", ""
	}
	runes, needleRunes := []rune(text), []rune(needle)
	for start := 0; start+len(needleRunes) <= len(runes); start++ {
		end := start + len(needleRunes)
		if strings.EqualFold(string(runes[start:end]), needle) {
			return string(runes[:start]), string(runes[start:end]), string(runes[end:])
		}
	}
	return text, "", ""
}
