package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// Most tests open one collection; these reach it the way the App did when
// it held a single collection.

// col is the first open collection.
func (a *App) col() *request.Collection { return a.open.Entries[0].Collection }

// workspace is the folder of the first open collection, "" when not saved.
func (a *App) workspace() string { return a.open.Entries[0].Dir }

// setWorkspace makes dir the folder of the first open collection.
func (a *App) setWorkspace(dir string) {
	a.open = collections.Update(a.open, collections.Located{Collection: a.col(), Dir: dir})
}

// envIndex is the index of the focused collection's active environment.
func (a *App) envIndex() int {
	entry, _ := a.focus()
	return entry.Env
}
