package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// treeMenuItems are the right-click actions for a row.
func (a *App) treeMenuItems(row collection.Row) []MenuItem {
	node := row.Node()
	rename := MenuItem{Label: "Rename", Run: func() { a.renameNode(node) }}
	sep := MenuItem{Separator: true}
	duplicate := MenuItem{Label: "Duplicate", Run: func() { a.duplicateNode(node) }}
	moveTo := MenuItem{Label: "Move to...", Run: func() { a.openTransferPicker(node, transfer.Move) }}
	copyTo := MenuItem{Label: "Copy to...", Run: func() { a.openTransferPicker(node, transfer.Copy) }}
	if row.Req != nil {
		return []MenuItem{
			rename, duplicate, moveTo, copyTo,
			{Label: "Copy URL", Run: func() { a.copyRequestURL(row.Req) }},
			sep,
			{Label: "Delete", Run: func() { a.deleteNode(node) }},
		}
	}
	create := []MenuItem{
		{Label: "New request", Run: func() { a.newRequestIn(row.Dir) }},
		{Label: "New folder", Run: func() { a.newFolderIn(row.Dir) }},
		{Label: "Import cURL from clipboard", Run: func() { a.importFolder(row.Dir, curlKind) }},
		{Label: "Import OpenAPI from clipboard", Run: func() { a.importFolder(row.Dir, openAPIKind) }},
		{Label: "Import Postman from clipboard", Run: func() { a.importFolder(row.Dir, postmanKind) }},
		{Label: "Export as Postman...", Run: func() { a.exportPostman(row.Dir) }},
		{Label: "Variables", Run: func() { a.editVars(row.Dir) }},
		sep,
		rename,
	}
	if a.isRoot(row.Dir) {
		return append(create[:len(create)-1], a.rootMenuItems(row.Dir)...)
	}
	return append(create, duplicate, moveTo, copyTo,
		MenuItem{Label: "Delete", Run: func() { a.deleteNode(node) }},
	)
}

// ---- operations ----------------------------------------------------------

// rootMenuItems end the menu of a collection's root row: the collection
// itself is renamed, moved, closed and deleted as a collection, not as a
// folder. Reordering is by menu rather than by drag, since a drag moves or
// copies a node into a collection.
func (a *App) rootMenuItems(root *request.Folder) []MenuItem {
	owner, _ := a.open.Owner(request.Node{Dir: root})
	col := owner.Collection
	at := a.openIndex(col)
	return []MenuItem{
		{Label: "Rename collection...", Run: func() { a.promptRenameCollection(col) }},
		{Label: "Move up", Disabled: at == 0, Run: func() { a.moveCollection(col, -1) }},
		{Label: "Move down", Disabled: at == len(a.open.Entries)-1, Run: func() { a.moveCollection(col, 1) }},
		menuSeparator,
		{Label: "New collection...", Run: a.promptNewCollection},
		{Label: "Open collection folder...", Run: a.promptWorkspace},
		menuSeparator,
		{Label: "Close collection", Run: func() { a.requestCloseCollection(col) }},
		{Label: "Delete collection...", Run: func() { a.confirmDeleteCollection(col) }},
	}
}
