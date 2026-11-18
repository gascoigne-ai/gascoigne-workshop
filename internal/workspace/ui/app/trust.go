package app

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// allowed reports whether source may be sent or connected now. A request
// that would run a command or read a file outside its collection, which an
// imported collection can ask for without the user knowing, first asks the
// user; the answer is remembered (state.Trusted) per command and per file.
// When it has to ask it returns false and runs again after the user
// allows.
func (a *App) allowed(tab *reqTab, source *model.Request, again func()) bool {
	folder := ""
	if entry, ok := a.entryFor(tab.Req); ok {
		folder = entry.Dir
	}
	var unknown []model.Access
	for _, access := range source.Expanded(a.varsFor(tab.Req)).Accesses(folder) {
		if !a.trusted.Has(access) {
			unknown = append(unknown, access)
		}
	}
	if len(unknown) == 0 {
		return true
	}
	var lines []string
	for _, access := range unknown {
		lines = append(lines, access.Kind+": "+access.Subject)
	}
	a.confirm("Allow this request?", "It wants to use:\n"+strings.Join(lines, "\n")+"\n\nOnly allow what you recognise. The answer is remembered.", "Allow", func() {
		for _, access := range unknown {
			a.trusted.Add(access)
		}
		again()
	})
	return false
}
