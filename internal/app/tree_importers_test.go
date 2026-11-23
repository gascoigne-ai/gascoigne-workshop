package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
)

// The tests call the three importers by their short names.
func (a *App) importCurl(dir *request.Folder)    { sidebarHost{a}.Import(dir, logic.ImportCurl) }
func (a *App) importOpenAPI(dir *request.Folder) { sidebarHost{a}.Import(dir, logic.ImportOpenAPI) }
func (a *App) importPostman(dir *request.Folder) { sidebarHost{a}.Import(dir, logic.ImportPostman) }
