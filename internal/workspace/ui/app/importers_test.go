package app

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// The tests call the three importers by their short names.
func (a *App) importCurl(dir *model.Folder)    { a.importFolder(dir, curlKind) }
func (a *App) importOpenAPI(dir *model.Folder) { a.importFolder(dir, openAPIKind) }
func (a *App) importPostman(dir *model.Folder) { a.importFolder(dir, postmanKind) }
