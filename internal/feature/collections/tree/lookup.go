package tree

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"

// PathSep joins folder and request names in a request's path.
const PathSep = "/"

// PathOf is the path of req below root, "" when it is not there. Of two
// requests with one path, the first in tree order has it.
func PathOf(root *request.Folder, req *request.Request) string {
	found := ""
	WalkRequests(root, "", func(path string, candidate *request.Request) {
		if candidate == req && found == "" {
			found = path
		}
	})
	return found
}

// At is the request of root at path, nil when there is none. Of two
// requests with one path, the first in tree order is returned.
func At(root *request.Folder, path string) *request.Request {
	var found *request.Request
	WalkRequests(root, "", func(candidate string, req *request.Request) {
		if candidate == path && found == nil {
			found = req
		}
	})
	return found
}

// WalkFolders calls visit for every folder below folder, with its path.
func WalkFolders(folder *request.Folder, prefix string, visit func(path string, folder *request.Folder)) {
	for _, sub := range folder.Folders {
		path := prefix + sub.Name
		visit(path, sub)
		WalkFolders(sub, path+PathSep, visit)
	}
}

// WalkRequests calls visit for every request below folder, with its path:
// the requests of a folder come before those of its subfolders.
func WalkRequests(folder *request.Folder, prefix string, visit func(path string, call *request.Request)) {
	for _, req := range folder.Requests {
		visit(prefix+req.Name, req)
	}
	for _, sub := range folder.Folders {
		WalkRequests(sub, prefix+sub.Name+PathSep, visit)
	}
}
