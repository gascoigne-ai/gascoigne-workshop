package auth

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// dialog is a question the pane put to the user.
type dialog struct {
	title, body, action string
	run                 func()
}

// toast is a message the pane showed.
type toast struct {
	text  string
	level severity.Level
}

// fake is every dependency of the pane at once: a real tab list and a real
// cookie jar, and a record of what the pane asked of the host.
type fake struct {
	set      *tabs.Set
	vars     []request.KV
	folder   string
	trusted  request.AccessSet
	jar      http.CookieJar
	clearErr error
	cleared  int
	dialogs  []dialog
	toasts   []toast
}

func (fake *fake) Current() *tabs.Tab { return fake.set.Current() }
func (fake *fake) IsOpen(tab *tabs.Tab) bool {
	return slices.Contains(fake.set.Tabs(), tab)
}

func (fake *fake) VarsFor(*request.Request) []request.KV { return fake.vars }
func (fake *fake) Folder(*request.Request) string        { return fake.folder }

func (fake *fake) Has(access request.Access) bool { return fake.trusted.Has(access) }
func (fake *fake) Add(access request.Access)      { fake.trusted.Add(access) }

func (fake *fake) Cookies(u *url.URL) []*http.Cookie { return fake.jar.Cookies(u) }
func (fake *fake) Clear() error {
	fake.cleared++
	return fake.clearErr
}

func (fake *fake) Confirm(title, body, action string, run func()) {
	fake.dialogs = append(fake.dialogs, dialog{title, body, action, run})
}
func (fake *fake) Toast(text string, level severity.Level) {
	fake.toasts = append(fake.toasts, toast{text, level})
}

// world is a pane over a fake, driven like the window does.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
}

// newWorld is a pane over one open request.
func newWorld(t *testing.T, req *request.Request) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet(), jar: jar}
	fake.set.Open(req)
	pane := New(Deps{Tabs: fake, Vars: fake, Trust: fake, Cookies: fake, Window: fake})
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: fake.set.Current()}
}

func TestTheAuthStateIsMadeFromTheRequestOnFirstUse(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Auth: request.Auth{Type: request.AuthBearer, Token: "abc"}})
	token := world.pane.Field(world.tab, requestedit.AuthToken)
	if token.Text != "abc" {
		t.Fatalf("token field %q, want abc", token.Text)
	}
	if world.pane.Field(world.tab, requestedit.AuthToken) != token {
		t.Fatal("the field must keep its place between calls: views hold pointers into it")
	}
}

func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "first"})
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second"})
	typed := world.pane.Field(first, requestedit.AuthToken)
	typed.SelectAll()
	typed.Insert("typed and not saved")
	if world.pane.Field(first, requestedit.AuthToken).Text != "typed and not saved" {
		t.Fatal("the field lost the text before the tab closed")
	}
	world.fake.set.Apply(tabs.Close{Index: 0})
	if world.pane.Tick() {
		t.Fatal("dropping a state changes nothing on screen")
	}
	if world.pane.Field(first, requestedit.AuthToken).Text != "" {
		t.Fatal("the closed tab's draft survived Tick")
	}
}

func TestTypingCountsOnlyTheCurrentTabsFields(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "first"})
	if world.pane.Typing() {
		t.Fatal("nothing is focused at the start")
	}
	if len(world.pane.states) != 0 {
		t.Fatal("asking whether the user types must not make state")
	}
	// A field of a type the pane does not show still holds the keyboard.
	world.pane.Field(world.tab, requestedit.AuthToken).Focused = true
	if !world.pane.Typing() {
		t.Fatal("a focused field of the current tab is typing")
	}
	world.fake.set.Open(&request.Request{Name: "second"})
	if world.pane.Typing() {
		t.Fatal("the focus of a hidden tab's field does not count")
	}
}

func TestToggleSecretsMasksEverySecretField(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Auth: request.Auth{Type: request.AuthBasic}})
	user := world.pane.Field(world.tab, requestedit.AuthUser)
	password := world.pane.Field(world.tab, requestedit.AuthPassword)
	if user.Masked || !password.Masked {
		t.Fatalf("user masked %v, password masked %v", user.Masked, password.Masked)
	}
	world.pane.toggleSecrets(world.tab)
	if user.Masked || password.Masked {
		t.Fatal("the toggle must show the secret")
	}
	world.pane.toggleSecrets(world.tab)
	if user.Masked || !password.Masked {
		t.Fatal("and hide it again")
	}
}

// stdioFile is a request that runs a command and reads a file.
func stdioFile() *request.Request {
	return &request.Request{Name: "R", Protocol: "mcp", URL: "stdio:/bin/true --flag", BodyType: request.BodyFile, Body: "/srv/data/body.json"}
}

func TestAllowedPassesKnownAccessesWithoutAsking(t *testing.T) {
	world := newWorld(t, stdioFile())
	world.fake.trusted.Add(request.Access{Kind: request.AccessCommand, Subject: "/bin/true --flag"})
	world.fake.trusted.Add(request.Access{Kind: request.AccessFile, Subject: "/srv/data/body.json"})
	if !world.pane.Allowed(world.tab, world.tab.Req, func() { t.Fatal("again runs only after a question") }) {
		t.Fatal("a request whose accesses are all allowed must pass")
	}
	if len(world.fake.dialogs) != 0 {
		t.Fatalf("asked %v", world.fake.dialogs)
	}
}

func TestAllowedLeavesFilesInsideTheCollectionAlone(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Protocol: "rest", BodyType: request.BodyFile, Body: "/srv/data/body.json"})
	world.fake.folder = "/srv/data"
	if !world.pane.Allowed(world.tab, world.tab.Req, func() {}) || len(world.fake.dialogs) != 0 {
		t.Fatalf("a file in the collection's folder asked: %v", world.fake.dialogs)
	}
}

func TestAllowedAsksOnceAndRemembersEveryUnknownAccess(t *testing.T) {
	world := newWorld(t, stdioFile())
	again := 0
	if world.pane.Allowed(world.tab, world.tab.Req, func() { again++ }) {
		t.Fatal("unknown accesses must not pass")
	}
	if len(world.fake.dialogs) != 1 {
		t.Fatalf("%d dialogs, want 1", len(world.fake.dialogs))
	}
	asked := world.fake.dialogs[0]
	wantBody := "It wants to use:\ncommand: /bin/true --flag\nfile: /srv/data/body.json\n\n" +
		"Only allow what you recognise. The answer is remembered."
	if asked.title != "Allow this request?" || asked.action != "Allow" || asked.body != wantBody {
		t.Fatalf("dialog %+v", asked)
	}
	if again != 0 || world.fake.trusted.Has(request.Access{Kind: request.AccessCommand, Subject: "/bin/true --flag"}) {
		t.Fatal("nothing is allowed before the user says yes")
	}
	asked.run()
	if again != 1 {
		t.Fatalf("again ran %d times, want 1", again)
	}
	if !world.pane.Allowed(world.tab, world.tab.Req, func() {}) || len(world.fake.dialogs) != 1 {
		t.Fatal("an allowed request asked again")
	}
}

func TestShowCookiesListsTheJarsCookiesForTheExpandedURL(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", URL: "https://{{host}}/x"})
	world.fake.vars = []request.KV{{Name: "host", Value: "api.test", Enabled: true}}
	world.pane.ShowCookies(world.tab)
	if len(world.fake.dialogs) != 1 || world.fake.dialogs[0].body != "No cookies for api.test." {
		t.Fatalf("dialogs %+v", world.fake.dialogs)
	}
	u, _ := url.Parse("https://api.test/x")
	world.fake.jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "abc", Path: "/"}})
	world.pane.ShowCookies(world.tab)
	asked := world.fake.dialogs[1]
	if asked.title != "Cookies" || asked.action != "Clear all cookies" || asked.body != "Sent to api.test:\nsession=abc" {
		t.Fatalf("dialog %+v", asked)
	}
}

func TestShowCookiesWithoutAHostToastsAnError(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", URL: "not a url"})
	world.pane.ShowCookies(world.tab)
	want := toast{"The URL has no host to look up cookies for", severity.Error}
	if len(world.fake.dialogs) != 0 || !slices.Equal(world.fake.toasts, []toast{want}) {
		t.Fatalf("dialogs %v toasts %v", world.fake.dialogs, world.fake.toasts)
	}
}

func TestClearingCookiesAsksTheHostAndToasts(t *testing.T) {
	cases := []struct {
		name  string
		fails error
		want  toast
	}{
		{"cleared", nil, toast{"Cookies cleared", severity.Info}},
		{"failed", errClear, toast{errClear.Error(), severity.Error}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t, &request.Request{Name: "R", URL: "https://api.test/x"})
			world.fake.clearErr = tc.fails
			world.pane.ShowCookies(world.tab)
			world.fake.dialogs[0].run()
			if world.fake.cleared != 1 || !slices.Equal(world.fake.toasts, []toast{tc.want}) {
				t.Fatalf("cleared %d toasts %v, want %v", world.fake.cleared, world.fake.toasts, tc.want)
			}
		})
	}
}

var errClear = errors.New("no jar for you")
