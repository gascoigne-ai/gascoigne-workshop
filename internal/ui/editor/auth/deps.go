package auth

import (
	"net/http"
	"net/url"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Deps is what the auth pane asks of the application, each a narrow
// interface the host implements.
type Deps struct {
	Tabs    Tabs
	Vars    Vars
	Trust   Trust
	Cookies Cookies
	Window  Window
}

// Tabs says which tab the pane shows and which are still open.
type Tabs interface {
	// Current is the tab shown, nil when none is open.
	Current() *tabs.Tab
	IsOpen(tab *tabs.Tab) bool
}

// Vars are the variables a request is resolved with and the folder of the
// collection it belongs to.
type Vars interface {
	VarsFor(req *request.Request) []request.KV
	// Folder is the directory of the collection that owns req, "" for none.
	Folder(req *request.Request) string
}

// Trust is what the user allowed, kept by the host in its state file.
type Trust interface {
	Has(access request.Access) bool
	Add(access request.Access)
}

// Cookies is the cookie jar the sends share.
type Cookies interface {
	// Cookies are the cookies the jar holds for u.
	Cookies(u *url.URL) []*http.Cookie
	// Clear forgets every cookie: the host replaces the jar and rebuilds
	// the protocols that hold it, so connections made before keep nothing.
	Clear() error
}

// Window is what the pane does to the rest of the window.
type Window interface {
	Confirm(title, body, action string, run func())
	Toast(text string, level severity.Level)
}
