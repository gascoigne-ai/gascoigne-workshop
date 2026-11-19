// Package transfer moves or copies a request or folder to another place in
// the same or another open collection: what is allowed, what a name clash
// does and which variables the node uses that its new place does not define.
// The tree and the picker draw it and ask the questions.
package transfer

import (
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Mode is what happens to the node at its old place.
type Mode int

// The two modes: Move takes the node out of its old place, Copy leaves it.
const (
	Move Mode = iota
	Copy
)

// Transfer is one move or copy. Node is in From and is not its root; Into
// is a folder of To (a collection's root folder for its top level) that is
// not inside Node. Before is a sibling in Into to land before, none for the
// end. A move into the node's own parent is a reorder. ToEnv is the active
// environment of To, nil for none; it only decides which variables count as
// missing.
type Transfer struct {
	Node, Before model.Node
	From, To     *model.Collection
	Into         *model.Folder
	Mode         Mode
	ToEnv        *model.Environment
}

// Outcome is what Apply did. Node is the node now in To: the moved one with
// its pointer, or the new copy. Name is its final name, Renamed says a
// sibling's name forced a suffix. Missing lists the variables Node uses that
// are not defined at its new place, in first-use order; variables are never
// rewritten. Err is set when the transfer was refused; nothing changed then.
type Outcome struct {
	Node    model.Node
	Name    string
	Renamed bool
	Missing []string
	Err     error
}

// Apply validates the transfer, then performs it. A refusal leaves both
// trees as they were.
func Apply(transfer Transfer) Outcome {
	if err := validate(transfer); err != nil {
		return Outcome{Err: err}
	}
	node := transfer.Node
	if transfer.Mode == Move && transfer.Before == node {
		return outcome(transfer, node, false) // dropped on itself: it stays where it is
	}
	if transfer.Mode == Copy {
		node = node.Clone()
	} else {
		transfer.From.Delete(node.Any())
	}
	renamed := transfer.Into.Attach(node, transfer.Before)
	return outcome(transfer, node, renamed)
}

func validate(transfer Transfer) error {
	from, to, node := transfer.From, transfer.To, transfer.Node
	switch {
	case from == nil || to == nil || node.None() || transfer.Into == nil:
		return model.ErrNotInTree
	case node.Dir == &from.Folder:
		return model.ErrRootMove
	case !from.Contains(node) || !to.Contains(model.Of(transfer.Into)):
		return model.ErrNotInTree
	case node.Dir != nil && node.Dir.Contains(model.Of(transfer.Into)):
		return model.ErrIntoItself
	}
	return nil
}

func outcome(transfer Transfer, node model.Node, renamed bool) Outcome {
	return Outcome{Node: node, Name: node.Name(), Renamed: renamed, Missing: missing(transfer, node)}
}

// missing is evaluated after the node sits in its new place, because the
// variables a request sees depend on the folders above it.
func missing(transfer Transfer, node model.Node) []string {
	var out []string
	seen := map[string]bool{}
	for _, req := range requestsOf(node) {
		vars := transfer.To.VarsFor(req, transfer.ToEnv)
		for _, name := range model.UsedVars(req) {
			if !seen[name] && !model.Defined(name, vars) {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// requestsOf lists the requests of node in tree order: a folder's own
// requests first, then those of its subfolders.
func requestsOf(node model.Node) []*model.Request {
	if node.Req != nil {
		return []*model.Request{node.Req}
	}
	out := append([]*model.Request(nil), node.Dir.Requests...)
	for _, sub := range node.Dir.Folders {
		out = append(out, requestsOf(model.Node{Dir: sub})...)
	}
	return out
}

// Drop is the Transfer of dropping node, of from, on target, a node of to:
// the place comes from the target's own collection. ToEnv is left for the
// caller to set.
func Drop(from, to *model.Collection, node, target model.Node, mode Mode) Transfer {
	into, before := tree.DropPlace(&to.Folder, node, target)
	return Transfer{Node: node, From: from, To: to, Into: into, Before: before, Mode: mode}
}

// Why a destination cannot be chosen; WhyNone means it can.
type Why int

// The reasons a destination is refused.
const (
	WhyNone       Why = iota
	WhyItself         // the folder is the node
	WhyInside         // the folder lies inside the node
	WhySameParent     // a move there changes nothing: the node is already in it
)

// Destination is one row of the picker: a folder of an open collection (the
// collection's root folder at Depth 0) and whether it may be chosen.
type Destination struct {
	Folder     *model.Folder
	Collection *model.Collection
	Depth      int
	Why        Why
}

// Destinations lists every folder of every open collection, collection by
// collection in tree order (a folder before its subfolders), with the reason
// node cannot go there for mode, if any.
func Destinations(set collections.Set, node model.Node, mode Mode) []Destination {
	var parent *model.Folder
	if owner, ok := set.Owner(node); ok {
		parent, _ = owner.Collection.Parent(node.Any())
	}
	var out []Destination
	for _, entry := range set.Entries {
		out = appendFolders(out, entry.Collection, &entry.Collection.Folder, 0, func(folder *model.Folder) Why {
			return reason(folder, node, parent, mode)
		})
	}
	return out
}

func appendFolders(out []Destination, col *model.Collection, folder *model.Folder, depth int, why func(*model.Folder) Why) []Destination {
	out = append(out, Destination{Folder: folder, Collection: col, Depth: depth, Why: why(folder)})
	for _, sub := range folder.Folders {
		out = appendFolders(out, col, sub, depth+1, why)
	}
	return out
}

func reason(folder *model.Folder, node model.Node, parent *model.Folder, mode Mode) Why {
	switch {
	case node.Dir == folder:
		return WhyItself
	case node.Dir != nil && node.Dir.Contains(model.Of(folder)):
		return WhyInside
	case mode == Move && parent == folder:
		return WhySameParent
	}
	return WhyNone
}

// maxNamedVars is how many missing variables MissingText names before it
// counts the rest.
const maxNamedVars = 5

// MissingText says which variables a moved or copied node uses that the
// collection called name does not define, naming at most maxNamedVars.
func MissingText(missing []string, name string) string {
	shown := missing
	if len(shown) > maxNamedVars {
		shown = shown[:maxNamedVars]
	}
	text := "Not defined in " + name + ": " + strings.Join(shown, ", ")
	if rest := len(missing) - len(shown); rest > 0 {
		text += fmt.Sprintf(" and %d more", rest)
	}
	return text
}
