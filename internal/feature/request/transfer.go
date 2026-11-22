package request

// Attach puts node, a request or folder that is in no tree (a clone, or one
// taken out with Delete), into f before the sibling `before` (a request for
// a request, a folder for a folder), at the end when before is none or not
// found. The pointer is kept. A name a sibling already has is made unique
// ("x 2"); the result reports whether that renamed node. Detach first, so a
// reorder inside one folder keeps its name.
func (f *Folder) Attach(node, before Node) (renamed bool) {
	field := nameOf(node)
	if field == nil {
		return false
	}
	unique := f.UniqueName(*field)
	renamed = unique != *field
	*field = unique
	switch {
	case node.Req != nil:
		at := len(f.Requests)
		if index := indexOf(f.Requests, before.Req); before.Req != nil && index >= 0 {
			at = index
		}
		f.Requests = insertAt(f.Requests, at, node.Req)
	case node.Dir != nil:
		at := len(f.Folders)
		if index := indexOf(f.Folders, before.Dir); before.Dir != nil && index >= 0 {
			at = index
		}
		f.Folders = insertAt(f.Folders, at, node.Dir)
	}
	return renamed
}

// Clone is a deep copy of the request or folder (with its variables and
// everything below it) under a new pointer; the clone of none is none.
func (n Node) Clone() Node {
	switch {
	case n.Req != nil:
		clone := n.Req.Clone()
		return Node{Req: &clone}
	case n.Dir != nil:
		return Node{Dir: n.Dir.deepCopy()}
	}
	return Node{}
}

// UsedVars lists the names of the {{variables}} req uses in its URL, enabled
// rows, bodies and credentials, in order of first use and without repeats.
func UsedVars(req *Request) []string {
	texts := []string{req.URL}
	for _, rows := range [][]KV{req.Params, req.Headers, req.Form} {
		for _, row := range rows {
			if row.Enabled {
				texts = append(texts, row.Name, row.Value)
			}
		}
	}
	auth := req.Auth
	texts = append(texts, req.Body, req.GraphQLVars, auth.Token, auth.User, auth.Password, auth.KeyName, auth.KeyValue)
	var names []string
	seen := map[string]bool{}
	for _, text := range texts {
		for _, name := range VarNames(text) {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}
