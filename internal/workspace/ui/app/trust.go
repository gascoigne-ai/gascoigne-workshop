package app

import (
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// allowed reports whether source may be sent or connected now. A request
// that would run a command or read a file outside its collection, which an
// imported collection can ask for without the user knowing, first asks the
// user; the answer is remembered (state.Trusted) per command and per file.
// When it has to ask it returns false and runs again after the user
// allows.
func (a *App) allowed(t *reqTab, source *model.Request, again func()) bool {
	folder := ""
	if entry, ok := a.entryFor(t.Req); ok {
		folder = entry.Dir
	}
	var unknown []model.Access
	for _, access := range source.Expanded(a.varsFor(t.Req)).Accesses(folder) {
		if !a.trusted[access.Key()] {
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
		if a.trusted == nil {
			a.trusted = map[string]bool{}
		}
		for _, access := range unknown {
			a.trusted[access.Key()] = true
		}
		again()
	})
	return false
}

// trustedKeys is the remembered answers in a stable order, for the state.
func (a *App) trustedKeys() []string {
	keys := make([]string, 0, len(a.trusted))
	for key := range a.trusted {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
