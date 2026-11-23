package sidebar

import (
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var separator = MenuItem{Separator: true}

// RowMenu is the right-click menu of a tree row; every item asks the
// application through Deps.
func (c *Component) RowMenu(row collection.Row) []MenuItem {
	node := row.Node()
	rename := MenuItem{Label: "Rename", Run: func() { c.deps.Nodes.Rename(node) }}
	duplicate := MenuItem{Label: "Duplicate", Run: func() { c.deps.Nodes.Duplicate(node) }}
	moveTo := MenuItem{Label: "Move to...", Run: func() { c.deps.Nodes.Pick(node, transfer.Move) }}
	copyTo := MenuItem{Label: "Copy to...", Run: func() { c.deps.Nodes.Pick(node, transfer.Copy) }}
	remove := MenuItem{Label: "Delete", Run: func() { c.deps.Nodes.Delete(node) }}
	if row.Req != nil {
		return []MenuItem{
			rename, duplicate, moveTo, copyTo,
			{Label: "Copy URL", Run: func() { c.deps.Extras.CopyURL(row.Req) }},
			separator,
			remove,
		}
	}
	create := []MenuItem{
		{Label: "New request", Run: func() { c.deps.Folders.NewRequest(row.Dir) }},
		{Label: "New folder", Run: func() { c.deps.Folders.NewFolder(row.Dir) }},
		{Label: "Import cURL from clipboard", Run: func() { c.deps.Folders.Import(row.Dir, logic.ImportCurl) }},
		{Label: "Import OpenAPI from clipboard", Run: func() { c.deps.Folders.Import(row.Dir, logic.ImportOpenAPI) }},
		{Label: "Import Postman from clipboard", Run: func() { c.deps.Folders.Import(row.Dir, logic.ImportPostman) }},
		{Label: "Export as Postman...", Run: func() { c.deps.Folders.Export(row.Dir) }},
		{Label: "Variables", Run: func() { c.deps.Extras.EditVars(row.Dir) }},
		separator,
		rename,
	}
	if c.deps.Collections.Set().IsRoot(row.Dir) {
		return append(create[:len(create)-1], c.rootMenu(row.Dir)...)
	}
	return append(create, duplicate, moveTo, copyTo, remove)
}

// rootMenu ends the menu of a collection's root row: the collection
// itself is renamed, moved, closed and deleted as a collection, not as a
// folder. Reordering is by menu rather than by drag, since a drag moves or
// copies a node into a collection.
func (c *Component) rootMenu(root *request.Folder) []MenuItem {
	set := c.deps.Collections.Set()
	owner, _ := set.Owner(request.Node{Dir: root})
	col := owner.Collection
	at := slices.IndexFunc(set.Entries, func(entry collections.Entry) bool { return entry.Collection == col })
	return []MenuItem{
		{Label: "Rename collection...", Run: func() { c.deps.Collection.Rename(col) }},
		{Label: "Move up", Disabled: at == 0, Run: func() { c.deps.Collection.Move(col, -1) }},
		{Label: "Move down", Disabled: at == len(set.Entries)-1, Run: func() { c.deps.Collection.Move(col, 1) }},
		separator,
		{Label: "New collection...", Run: c.deps.Workspace.NewCollection},
		{Label: "Open collection folder...", Run: c.deps.Workspace.OpenFolder},
		separator,
		{Label: "Close collection", Run: func() { c.deps.Collection.Close(col) }},
		{Label: "Delete collection...", Run: func() { c.deps.Collection.Delete(col) }},
	}
}

// CollectionMenu is the menu of the Collections chip, which acts on the
// focused collection; with nothing open only New and Open act.
func (c *Component) CollectionMenu() []MenuItem {
	col := c.deps.Collections.Focused()
	none := col == nil
	return []MenuItem{
		{Label: "New collection...", Run: c.deps.Workspace.NewCollection},
		{Label: "Open collection folder...", Run: c.deps.Workspace.OpenFolder},
		separator,
		{Label: "Rename collection...", Disabled: none, Run: func() { c.deps.Collection.Rename(col) }},
		{Label: "Save collection", Disabled: none, Run: func() { c.deps.Workspace.Save(col) }},
		separator,
		{Label: "Close collection", Disabled: none, Run: func() { c.deps.Collection.Close(col) }},
		{Label: "Delete collection...", Disabled: none, Run: func() { c.deps.Collection.Delete(col) }},
	}
}
