package app

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"

// The tests call the three importers by their short names.
func (a *App) importCurl(dir *request.Folder)    { a.importFolder(dir, curlKind) }
func (a *App) importOpenAPI(dir *request.Folder) { a.importFolder(dir, openAPIKind) }
func (a *App) importPostman(dir *request.Folder) { a.importFolder(dir, postmanKind) }
