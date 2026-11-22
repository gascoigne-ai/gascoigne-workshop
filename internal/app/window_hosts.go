package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/theme"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Size is the window's size in logical pixels; with the methods below it
// makes the App the host of the features that live in their own packages.
func (a *App) Size() (width, height int) { return a.w, a.h }

// SentStore, Toast (below), SetStatus, OpenRequest and Badge make the App
// the host of the sent-request log.
func (a *App) SentStore() history.SentStore {
	store := a.historyStore()
	if store == nil {
		return nil
	}
	return store.Sent()
}

// Toast shows a message over the window.
func (a *App) Toast(text string, level severity.Level) { a.toast(text, level) }

// SetStatus replaces the status line.
func (a *App) SetStatus(text string) { a.status = text }

// OpenRequest focuses the request at path of the collection saved in
// folder and reports whether there is one. A row without a folder is of a
// collection not saved, or older than the folder in a row, when the one
// collection open was the first.
func (a *App) OpenRequest(folder, path string) bool {
	for _, entry := range a.sentCandidates(folder) {
		if req := tree.At(&entry.Collection.Folder, path); req != nil {
			a.openRequest(req)
			return true
		}
	}
	return false
}

// sentCandidates are the open collections a sent row of folder may name,
// in the order they are searched.
func (a *App) sentCandidates(folder string) []collections.Entry {
	if folder != "" {
		entry, ok := a.open.ByDir(folder)
		if !ok {
			return nil
		}
		return []collections.Entry{entry}
	}
	var candidates []collections.Entry
	for _, entry := range a.open.Entries {
		if entry.Dir == "" {
			candidates = append(candidates, entry)
		}
	}
	if len(a.open.Entries) > 0 {
		candidates = append(candidates, a.open.Entries[0])
	}
	return candidates
}

// Badge is the method label shown before a request's name.
func (a *App) Badge(protocol request.Protocol, method string) Node {
	return Text(requestBadge(protocol, method), Small(), Color(theme.MethodToken(method)), W(badgeSlot(a.ctx, badgeW)))
}
