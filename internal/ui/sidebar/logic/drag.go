package logic

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// dragExpandTicks is how long a drag must rest over a closed folder
// before the folder opens, so something can be dropped inside it.
const dragExpandTicks = 30

// Drag is the press on a tree row, from the button going down to its
// release. It is one value, not a flag per fact: what it is doing is
// Phase, computed from the fields, so no "dropping" flag can disagree with
// them. A release is resolved in the frame it arrives (Release), so there
// is no stored state for it.
type Drag struct {
	// Node is the node being dragged, none when nothing is pressed or the
	// press was on a collection's root, which cannot be moved.
	Node request.Node
	// Pressed is the folder row the button went down on, toggled on release
	// there. A root press has it set with Node none, so it still toggles.
	Pressed *request.Folder
	// Target is the node the pointer is over, none when it is over no row.
	Target request.Node
	// Hover is the closed folder the drag has rested over, for Ticks frames.
	Hover *request.Folder
	Ticks int
}

// Phase is what a Drag is doing.
type Phase int

// The phases of a drag.
const (
	Idle     Phase = iota // nothing is pressed
	Pressed               // pressed, over no row or still over its own: a click if released now
	Dragging              // over another row: released now, it drops
)

// Phase tells what the drag is doing now. A root press is never Dragging,
// since a root cannot be moved.
func (drag Drag) Phase() Phase {
	switch {
	case drag.Node.None() && drag.Pressed == nil:
		return Idle
	case drag.Node.None() || drag.Target.None() || drag.Target == drag.Node:
		return Pressed
	}
	return Dragging
}

// OutcomeKind is what a release asks the caller to do.
type OutcomeKind int

// The outcomes of a release.
const (
	None   OutcomeKind = iota // nothing
	Toggle                    // a click on a folder row: the tree opened or closed it
	Drop                      // the node is dropped on the target
)

// Outcome is the result of a release.
type Outcome struct {
	Kind   OutcomeKind
	Node   request.Node  // the dragged node, for Drop
	Target request.Node  // the node it was released over, for Drop
	Mode   transfer.Mode // Move, or Copy when Alt was held at the release
}

// Drag is the drag in progress.
func (tree *Tree) Drag() Drag { return tree.drag }

// Press is the button going down on row. It selects the row, gives the tree
// the keyboard and starts a drag; a collection's root (isRoot) cannot be
// dragged, so it presses without a node. A folder toggles when the button is
// released on the same row (see Release), so an open folder can still be
// dragged; opening a request is the caller's.
func (tree *Tree) Press(row collection.Row, isRoot bool) {
	tree.focused = true
	tree.drag.Node = row.Node()
	if isRoot {
		tree.drag.Node = request.Node{}
	}
	tree.selected = row.Node()
	if row.Folder {
		tree.drag.Pressed = row.Dir
	}
}

// Over records that the pointer is over node while the button is held.
func (tree *Tree) Over(node request.Node) { tree.drag.Target = node }

// Clear forgets the node the pointer was over, on a frame where the button
// is held: the row under the pointer sets it again while the view updates.
func (tree *Tree) Clear() { tree.drag.Target = request.Node{} }

// Release is the button coming up: it ends the drag and says what it means.
// A release on the folder row the button went down on toggles that folder
// (done here). Over another row it is a drop of the dragged node, a move or,
// with alt, a copy; the caller checks that both nodes are in open
// collections. Over no row, or on a request's own row, it is nothing.
func (tree *Tree) Release(alt bool) Outcome {
	node, target, pressed := tree.drag.Node, tree.drag.Target, tree.drag.Pressed
	tree.drag.Node, tree.drag.Target, tree.drag.Pressed = request.Node{}, request.Node{}, nil
	if pressed != nil && target == (request.Node{Dir: pressed}) {
		tree.Toggle(pressed)
		return Outcome{Kind: Toggle}
	}
	if node.None() || target.None() || target == node {
		return Outcome{}
	}
	mode := transfer.Move
	if alt {
		mode = transfer.Copy
	}
	return Outcome{Kind: Drop, Node: node, Target: target, Mode: mode}
}

// Tick counts a frame of a drag resting over a closed folder and opens the
// folder at dragExpandTicks. It reports whether it opened one. The dragged
// folder itself, a drag with no node and an open folder reset the count.
func (tree *Tree) Tick() bool {
	folder := tree.drag.Target.Dir
	if tree.drag.Node.None() || folder == nil || tree.expanded[folder] || tree.drag.Node.Dir == folder {
		tree.drag.Hover, tree.drag.Ticks = nil, 0
		return false
	}
	if tree.drag.Hover != folder {
		tree.drag.Hover, tree.drag.Ticks = folder, 0
	}
	if tree.drag.Ticks++; tree.drag.Ticks < dragExpandTicks {
		return false
	}
	tree.expanded[folder] = true
	tree.drag.Hover, tree.drag.Ticks = nil, 0
	return true
}
