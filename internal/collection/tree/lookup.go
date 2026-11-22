package tree

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// PathSep joins folder and request names in a request's path.
const PathSep = "/"

// PathOf is the path of req below root, "" when it is not there. Of two
// requests with one path, the first in tree order has it.
func PathOf(root *model.Folder, req *model.Request) string {
	found := ""
	WalkRequests(root, "", func(path string, candidate *model.Request) {
		if candidate == req && found == "" {
			found = path
		}
	})
	return found
}

// At is the request of root at path, nil when there is none. Of two
// requests with one path, the first in tree order is returned.
func At(root *model.Folder, path string) *model.Request {
	var found *model.Request
	WalkRequests(root, "", func(candidate string, req *model.Request) {
		if candidate == path && found == nil {
			found = req
		}
	})
	return found
}

// WalkFolders calls visit for every folder below folder, with its path.
func WalkFolders(folder *model.Folder, prefix string, visit func(path string, folder *model.Folder)) {
	for _, sub := range folder.Folders {
		path := prefix + sub.Name
		visit(path, sub)
		WalkFolders(sub, path+PathSep, visit)
	}
}

// WalkRequests calls visit for every request below folder, with its path:
// the requests of a folder come before those of its subfolders.
func WalkRequests(folder *model.Folder, prefix string, visit func(path string, request *model.Request)) {
	for _, req := range folder.Requests {
		visit(prefix+req.Name, req)
	}
	for _, sub := range folder.Folders {
		WalkRequests(sub, prefix+sub.Name+PathSep, visit)
	}
}
