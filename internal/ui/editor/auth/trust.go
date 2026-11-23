package auth

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Allowed reports whether source may be sent or connected now. A request
// that would run a command or read a file outside its collection, which an
// imported collection can ask for without the user knowing, first asks the
// user; the answer is remembered (state.Trusted) per command and per file.
// When it has to ask it returns false and runs again after the user
// allows.
//
// It asks rather than refuses (the user's decision), because the file
// browser stores absolute paths and refusing them would break attaching a
// file. Not asked about: a .workshop.* config in the working directory (the
// user's decision) and .proto imports, which cannot leave their import
// paths.
func (c *Component) Allowed(owner *tabs.Tab, source *request.Request, again func()) bool {
	var unknown []request.Access
	folder := c.deps.Vars.Folder(owner.Req)
	for _, access := range source.Expanded(c.deps.Vars.VarsFor(owner.Req)).Accesses(folder) {
		if !c.deps.Trust.Has(access) {
			unknown = append(unknown, access)
		}
	}
	if len(unknown) == 0 {
		return true
	}
	var lines []string
	for _, access := range unknown {
		lines = append(lines, string(access.Kind)+": "+access.Subject)
	}
	body := "It wants to use:\n" + strings.Join(lines, "\n") + "\n\nOnly allow what you recognise. The answer is remembered."
	c.deps.Window.Confirm("Allow this request?", body, "Allow", func() {
		for _, access := range unknown {
			c.deps.Trust.Add(access)
		}
		again()
	})
	return false
}
