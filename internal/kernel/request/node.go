package request

// Node is a folder or a request of the tree; the zero Node is none. At most
// one field is set, so a typed nil pointer and no node are the same Node.
type Node struct {
	Req *Request
	Dir *Folder
}

// Of is the Node for a *Request or *Folder, the zero Node for anything else.
// It is the one place that looks at the dynamic type of a tree node, for a
// payload that crosses a generic boundary; code that holds a *Request or a
// *Folder writes Node{Req: ...} or Node{Dir: ...}.
func Of(node any) Node {
	switch n := node.(type) {
	case *Request:
		return Node{Req: n}
	case *Folder:
		return Node{Dir: n}
	}
	return Node{}
}

// Name is the name of the request or folder, "" for none.
func (n Node) Name() string {
	switch {
	case n.Req != nil:
		return n.Req.Name
	case n.Dir != nil:
		return n.Dir.Name
	}
	return ""
}

// None reports whether n stands for nothing.
func (n Node) None() bool { return n.Req == nil && n.Dir == nil }
