package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
)

// The tests call the three importers by their short names.
func (a *Component) importCurl(dir *request.Folder) { sidebarHost{a}.Import(dir, logic.ImportCurl) }
func (a *Component) importOpenAPI(dir *request.Folder) {
	sidebarHost{a}.Import(dir, logic.ImportOpenAPI)
}
func (a *Component) importPostman(dir *request.Folder) {
	sidebarHost{a}.Import(dir, logic.ImportPostman)
}
