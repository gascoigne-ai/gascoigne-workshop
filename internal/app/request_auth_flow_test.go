package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// recordServer remembers the last request it served.
type recorded struct {
	header  http.Header
	query   string
	user    string
	pass    string
	hasAuth bool
}

func recordServer(t *testing.T) (*httptest.Server, func() recorded) {
	t.Helper()
	var mu sync.Mutex
	var last recorded
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		last.header, last.query = call.Header.Clone(), call.URL.RawQuery
		last.user, last.pass, last.hasAuth = call.BasicAuth()
	}))
	t.Cleanup(srv.Close)
	return srv, func() recorded { mu.Lock(); defer mu.Unlock(); return last }
}

func TestBearerAuthSendsTheToken(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	click(a, authBearerX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("abc")
	req := a.cur().Req
	if req.Auth.Type != request.AuthBearer || req.Auth.Token != "abc" {
		t.Fatalf("auth = %+v", req.Auth)
	}
	if len(req.Headers) != 0 {
		t.Fatalf("credentials must not appear in the Headers table: %+v", req.Headers)
	}
	sendAndWait(t, a, srv.URL)
	if got := last().header.Get("Authorization"); got != "Bearer abc" {
		t.Fatalf("Authorization = %q", got)
	}
	if !dirty(a.cur()) {
		t.Fatal("typing credentials is a change")
	}
}

func TestBasicAuthSendsUserAndPassword(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	click(a, authBasicX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("ann")
	click(a, authFieldX, authRow2Y)
	a.typeText("pw")
	sendAndWait(t, a, srv.URL)
	if r := last(); !r.hasAuth || r.user != "ann" || r.pass != "pw" {
		t.Fatalf("basic auth = %+v", r)
	}
}

func TestAPIKeyGoesInTheHeaderOrTheQuery(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	click(a, authKeyX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("X-Key")
	click(a, authFieldX, authRow2Y)
	a.typeText("v1")

	sendAndWait(t, a, srv.URL)
	if got := last().header.Get("X-Key"); got != "v1" {
		t.Fatalf("header X-Key = %q", got)
	}

	click(a, 516, authRow3Y) // the Query choice
	if a.cur().Req.Auth.KeyIn != request.KeyInQuery {
		t.Fatalf("KeyIn = %q", a.cur().Req.Auth.KeyIn)
	}
	sendAndWait(t, a, srv.URL)
	if r := last(); r.query != "X-Key=v1" || r.header.Get("X-Key") != "" {
		t.Fatalf("query %q header %q", r.query, r.header.Get("X-Key"))
	}
}

func TestVariablesInCredentialsAndHeadersAreExpandedAtSend(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	a.col().Vars = append(a.col().Vars, request.KV{Name: "tok", Value: "sekret", Enabled: true})
	a.cur().Req.Headers = []request.KV{{Name: "X-Token", Value: "{{tok}}", Enabled: true}}
	click(a, authBearerX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("{{tok}}")
	sendAndWait(t, a, srv.URL)
	r := last()
	if r.header.Get("Authorization") != "Bearer sekret" || r.header.Get("X-Token") != "sekret" {
		t.Fatalf("headers %v", r.header)
	}
	if a.cur().Req.Auth.Token != "{{tok}}" {
		t.Fatal("the stored request keeps the variable")
	}
}
