package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProtocolAndMethodChipsCycle(t *testing.T) {
	a := testApp(t)
	a.settings.GRPC, a.settings.SSE = true, true // the full cycle takes both experimental protocols
	req := a.cur().Req

	click(a, 380, urlBarY) // method chip
	if req.Method != "POST" {
		t.Fatalf("method = %q, want POST", req.Method)
	}
	click(a, 300, urlBarY) // protocol chip
	if req.Protocol != "websocket" {
		t.Fatalf("protocol = %q, want websocket", req.Protocol)
	}
	click(a, 300, urlBarY)
	click(a, 300, urlBarY)
	click(a, 300, urlBarY)
	click(a, 300, urlBarY)
	if req.Protocol != "rest" || req.Method != "POST" {
		t.Fatalf("after a full cycle: %q %q", req.Protocol, req.Method)
	}
}

func TestSwitchingToRESTRestoresAMethod(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	req.Protocol, req.Method = "grpc", ""
	a.cycleProtocol(a.cur())
	if req.Protocol != "rest" || req.Method != "GET" {
		t.Fatalf("got %q %q, want rest GET", req.Protocol, req.Method)
	}
}

// The server holds its answer until the test has seen the request
// running. Without the hold, a fast answer could be applied by the
// second step of the click, so "Sending" was already false and the
// test failed when the machine was busy.
func TestSendShowsTheResponse(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-release
		io.WriteString(writer, `{"ok":true}`)
	}))
	defer srv.Close()
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	click(a, sendX, urlBarY)
	if !tab.Sending() {
		t.Fatal("a click on Send must start a request")
	}
	close(release)
	waitForResponse(t, a, tab)

	if tab.Sending() || tab.Resp.Status != 200 || tab.Resp.Err != "" {
		t.Fatalf("sending %v resp %+v", tab.Sending(), tab.Resp)
	}
	if len(tab.view.Lines) != 3 { // pretty-printed {"ok":true}
		t.Fatalf("response view has %d lines", len(tab.view.Lines))
	}
	if !strings.Contains(a.status, "200 OK") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestEnterInTheURLFieldSends(t *testing.T) {
	srv := jsonServer(t)
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	click(a, 1100, urlBarY) // focus the field
	a.step(core.Input{Enter: true})
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 200 {
		t.Fatalf("resp = %+v", tab.Resp)
	}
}

// A second send while one is running (the shortcut, since a click on the
// button now cancels) must not start another request.
func TestSendingTwiceWhileRunningStartsOneRequest(t *testing.T) {
	hits := make(chan struct{}, 4)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits <- struct{}{}
		<-release
	}))
	defer srv.Close()
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	click(a, sendX, urlBarY)
	a.step(combo(ebiten.KeyEnter, true, false, false))
	a.step(combo(ebiten.KeyEnter, true, false, false))
	<-hits
	close(release)
	waitForResponse(t, a, tab)
	if len(hits) != 0 {
		t.Fatal("the second click must not start another request")
	}
}

func TestFailedSendShowsTheError(t *testing.T) {
	srv := jsonServer(t)
	url := srv.URL
	srv.Close() // nothing listens any more
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, url)

	click(a, sendX, urlBarY)
	waitForResponse(t, a, tab)
	if tab.Resp.Err == "" || a.status != "Request failed" {
		t.Fatalf("err %q status %q", tab.Resp.Err, a.status)
	}
}

func TestOtherProtocolsAnswerThroughTheSameUI(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	click(a, 300, urlBarY)   // rest -> websocket
	click(a, sendX, urlBarY) // connect: an https URL is no WebSocket address
	until(t, a, "failed connect", func() bool {
		return tab.State() != tabs.ConnConnecting && strings.Contains(logText(tab), "not ws or wss")
	})
	if tab.Session() != nil {
		t.Fatal("connected to an https URL")
	}
	click(a, 300, urlBarY) // websocket -> mcp
	if tab.Req.Protocol != "mcp" {
		t.Fatalf("protocol %q", tab.Req.Protocol)
	}
	tab.Req.URL = "ftp://nowhere.test" // no network in tests
	click(a, sendX, urlBarY)
	until(t, a, "failed connect", func() bool {
		return tab.State() != tabs.ConnConnecting && strings.Contains(logText(tab), "not an http or https URL")
	})
}

// Every pane of both areas, with and without a response, must build and
// lay out; a pane that reads a nil response would panic here.
func TestEveryPaneBuildsInEveryResponseState(t *testing.T) {
	srv := jsonServer(t)
	a := testApp(t)
	tab := a.cur()

	states := map[string]func(){
		"no response": func() { tab.Resp = nil; setSending(tab, false) },
		"sending":     func() { tab.Resp = nil; setSending(tab, true) },
		"error":       func() { tab.Resp = &respErr; setSending(tab, false) },
		"ok": func() {
			setSending(tab, false)
			pointTabAt(tab, srv.URL)
			a.send(tab)
			waitForResponse(t, a, tab)
		},
	}
	for _, name := range []string{"no response", "sending", "error", "ok"} {
		states[name]()
		for i := range requestPanes {
			for j := range responsePanes {
				tab.paneTab, tab.respTab = i, j
				a.step(core.Input{}) // panics on a bad pane
			}
		}
	}
}

func sendingTab() *tabs.Tab {
	tab := &tabs.Tab{}
	tab.BeginSend()
	return tab
}

func TestNoResponseStates(t *testing.T) {
	tests := []struct {
		name    string
		tab     reqTab
		wantMsg string
		wantErr bool
	}{
		{"sending", reqTab{Tab: sendingTab()}, "Sending...", false},
		{"not sent", reqTab{Tab: &tabs.Tab{}}, "Send the request to see the response", false},
		{"failed", reqTab{Tab: &tabs.Tab{Resp: &respErr}}, "boom", true},
		{"ok", reqTab{Tab: &tabs.Tab{Resp: &respOK}}, "", false},
	}
	for _, tt := range tests {
		msg, isErr := noResponse(&tt.tab)
		if msg != tt.wantMsg || isErr != tt.wantErr {
			t.Errorf("%s: got %q %v", tt.name, msg, isErr)
		}
	}
}

func TestCtrlEnterSendsTheRequest(t *testing.T) {
	srv := jsonServer(t)
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.step(combo(ebiten.KeyEnter, true, false, false))
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 200 {
		t.Fatalf("resp = %+v", tab.Resp)
	}
}

func TestBodyTypeChipsSwitchTheTypeAndKeepTheText(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	if req.BodyType != model.BodyNone {
		t.Fatalf("a request without a body starts as none, got %q", req.BodyType)
	}
	click(a, textChipX, chipsY)
	if req.BodyType != model.BodyText {
		t.Fatalf("type = %q", req.BodyType)
	}
	click(a, editorX, editorY)
	a.typeText("hello")
	click(a, jsonChipX, chipsY)
	click(a, formChipX, chipsY)
	click(a, textChipX, chipsY)
	if req.Body != "hello" {
		t.Fatalf("the text must survive switching types, Body = %q", req.Body)
	}
	for chip, want := range map[int]model.BodyType{noneChipX: model.BodyNone, multiChipX: model.BodyMultipart, formChipX: model.BodyForm} {
		click(a, chip, chipsY)
		if req.BodyType != want {
			t.Errorf("chip at %d gave %q, want %q", chip, req.BodyType, want)
		}
	}
}

func TestTypingInTheBodyEditor(t *testing.T) {
	a := openBodyPane(t)
	click(a, jsonChipX, chipsY)
	click(a, editorX, editorY)
	a.typeText(`{"a":1}`)
	a.step(core.Input{Enter: true})
	a.typeText("x")
	if got := a.cur().Req.Body; got != "{\"a\":1}\nx" {
		t.Fatalf("Body = %q", got)
	}
}

func TestFormatButtonIndentsValidJSONOnly(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	click(a, jsonChipX, chipsY)
	req.Body = `{"a":1,"b":[1,2]}`
	a.cur().body.SelectAll()
	a.cur().body.Insert(req.Body)
	req.Body = a.cur().body.Text

	click(a, formatX, chipsY)
	if want := "{\n  \"a\": 1,\n  \"b\": [\n    1,\n    2\n  ]\n}"; req.Body != want {
		t.Fatalf("Body = %q", req.Body)
	}

	req.Body = `{"a":`
	a.cur().body.SelectAll()
	a.cur().body.Insert(req.Body)
	click(a, formatX, chipsY)
	if req.Body != `{"a":` || !strings.Contains(a.status, "Cannot format") {
		t.Fatalf("Body %q status %q: invalid JSON must be left alone and reported", req.Body, a.status)
	}
}

func TestFormatButtonOnlyExistsForJSON(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	click(a, textChipX, chipsY)
	req.Body = `{"a":1}`
	a.cur().body.SelectAll()
	a.cur().body.Insert(req.Body)
	click(a, formatX, chipsY)
	if req.Body != `{"a":1}` {
		t.Fatalf("a text body must not be reformatted, Body = %q", req.Body)
	}
}

func TestFormBodyFieldsAreEditable(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	click(a, formChipX, chipsY)
	click(a, nameCellX, formRow0Y)
	a.typeText("user")
	a.step(core.Input{Tab: true})
	a.typeText("ann")
	a.step(core.Input{Enter: true})
	if len(req.Form) != 1 || req.Form[0].Name != "user" || req.Form[0].Value != "ann" || !req.Form[0].Enabled {
		t.Fatalf("Form = %+v", req.Form)
	}
	if req.Body != "" {
		t.Fatalf("form fields must not touch the text body, Body = %q", req.Body)
	}
}

func TestSendingAFormBodyUsesTheRows(t *testing.T) {
	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		b, _ := io.ReadAll(request.Body)
		got <- request.Header.Get("Content-Type") + "|" + string(b)
	}))
	defer srv.Close()
	a := testApp(t)
	tab := a.cur()
	tab.Req.Method, tab.Req.BodyType = "POST", model.BodyForm
	tab.Req.Form = []model.KV{{Name: "q", Value: "go lang", Enabled: true}}
	pointTabAt(tab, srv.URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	if want := "application/x-www-form-urlencoded|q=go+lang"; <-got != want {
		t.Fatal("the sent request must carry the form body")
	}
}

func TestEachTabKeepsItsOwnBodyEditor(t *testing.T) {
	a := openBodyPane(t)
	click(a, textChipX, chipsY)
	click(a, editorX, editorY)
	a.typeText("first")
	a.newRequest()
	second := a.cur()
	second.paneTab = bodyTabIdx
	a.step(core.Input{})
	click(a, textChipX, chipsY)
	click(a, editorX, editorY)
	a.typeText("second")
	a.switchTab(-1)
	if a.cur().Req.Body != "first" || second.Req.Body != "second" {
		t.Fatalf("bodies %q and %q", a.cur().Req.Body, second.Req.Body)
	}
}

func TestSendingTurnsIntoCancelAndStopsTheRequest(t *testing.T) {
	srv, _ := pathServer(t)
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/slow")

	click(a, sendX, urlBarY)
	if !tab.Sending() {
		t.Fatal("test setup: the request must be running")
	}
	click(a, sendX, urlBarY) // the same button now cancels
	if tab.Sending() || tab.Resp == nil || tab.Resp.Err != "Cancelled" {
		t.Fatalf("sending %v resp %+v", tab.Sending(), tab.Resp)
	}
	if a.status != "Cancelled" {
		t.Fatalf("status = %q", a.status)
	}
}

func TestEscapeCancelsARunningRequestButClosesTheMenuFirst(t *testing.T) {
	srv, _ := pathServer(t)
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/slow")
	click(a, sendX, urlBarY)

	a.menu = true
	a.step(core.Input{Escape: true})
	if a.menu || !tab.Sending() {
		t.Fatalf("the first Escape closes the menu only: menu %v sending %v", a.menu, tab.Sending())
	}
	a.step(core.Input{Escape: true})
	if tab.Sending() || tab.Resp == nil || tab.Resp.Err != "Cancelled" {
		t.Fatalf("the second Escape cancels: sending %v resp %+v", tab.Sending(), tab.Resp)
	}
}

func TestEscapeWithNothingRunningDoesNothing(t *testing.T) {
	a := testApp(t)
	a.step(core.Input{Escape: true})
	if a.cur().Resp != nil {
		t.Fatal("Escape must not invent a response")
	}
}

func TestALateResultFromACancelledRequestIsIgnored(t *testing.T) {
	srv, release := pathServer(t)
	a := testApp(t)
	tab := a.cur()

	pointTabAt(tab, srv.URL+"/slow")
	click(a, sendX, urlBarY)
	click(a, sendX, urlBarY) // cancel
	pointTabAt(tab, srv.URL+"/fast")
	click(a, sendX, urlBarY)
	waitForResponse(t, a, tab)
	for tab.Resp.Err != "" || string(tab.Resp.Body) != "fast" { // the cancelled one may still be in flight
		a.step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}

	release()
	for range 40 {
		a.step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}
	if tab.Resp.Err != "" || string(tab.Resp.Body) != "fast" {
		t.Fatalf("the cancelled request's late result replaced the new one: %+v", tab.Resp)
	}
}

func TestRequestTimeoutStopsASlowRequest(t *testing.T) {
	srv, _ := pathServer(t)
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/slow")
	tab.Req.TimeoutSec = 1

	start := time.Now()
	click(a, sendX, urlBarY)
	waitForResponse(t, a, tab)
	if tab.Resp.Err != "Timed out" || time.Since(start) > 4*time.Second {
		t.Fatalf("resp %+v after %v", tab.Resp, time.Since(start))
	}
}

func TestTimeoutFallsBackToTheSettingsDefault(t *testing.T) {
	a := testApp(t)
	tests := []struct {
		name         string
		req, setting int
		want         time.Duration
	}{
		{"the request's own value wins", 5, 60, 5 * time.Second},
		{"zero uses the default", 0, 30, 30 * time.Second},
		{"no default either uses a minute", 0, 0, 60 * time.Second},
		{"a negative value is not a timeout", -3, 30, 30 * time.Second},
	}
	for _, tt := range tests {
		a.settings.TimeoutSec = tt.setting
		if got := a.timeoutFor(model.Request{TimeoutSec: tt.req}); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTimeoutFieldAcceptsWholeSecondsOnly(t *testing.T) {
	a := openSettingsPane(t)
	req := a.cur().Req
	click(a, timeoutFieldX, chipsY)
	a.typeText("7")
	if req.TimeoutSec != 7 {
		t.Fatalf("TimeoutSec = %d", req.TimeoutSec)
	}
	a.typeText("x")
	if req.TimeoutSec != 7 {
		t.Fatalf("a non-numeric edit must not change the timeout, got %d", req.TimeoutSec)
	}
	a.step(core.Input{Edits: []core.Edit{core.EditBackspace, core.EditBackspace}}) // clears the field
	if req.TimeoutSec != 0 {
		t.Fatalf("an empty field means the default, got %d", req.TimeoutSec)
	}
}

func TestFollowRedirectsToggleControlsRedirects(t *testing.T) {
	srv, _ := pathServer(t)
	a := openSettingsPane(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/redirect")

	a.send(tab)
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 200 {
		t.Fatalf("redirects are followed by default, status %d", tab.Resp.Status)
	}

	click(a, followRowX, followRowY)
	if !tab.Req.NoRedirects {
		t.Fatal("the checkbox row must turn following off")
	}
	tab.Resp = nil
	a.send(tab)
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 302 {
		t.Fatalf("redirects are not followed now, status %d", tab.Resp.Status)
	}
}

func TestRawAndPrettyToggleTheResponseView(t *testing.T) {
	a := jsonResponse(t, `{"a":1,"b":[2]}`)
	tab := a.cur()
	if len(tab.view.Lines) != 6 {
		t.Fatalf("pretty view has %d lines", len(tab.view.Lines))
	}
	click(a, rawX, toolbarY)
	if !tab.look.Raw() || len(tab.view.Lines) != 1 || tab.view.AllText() != `{"a":1,"b":[2]}` {
		t.Fatalf("raw: %d lines %q", len(tab.view.Lines), tab.view.AllText())
	}
	click(a, prettyX, toolbarY)
	if tab.look.Raw() || len(tab.view.Lines) != 6 {
		t.Fatalf("back to pretty: raw %v, %d lines", tab.look.Raw(), len(tab.view.Lines))
	}
}

func TestBinaryBodiesAreNotShownAsText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte{0x89, 'P', 'N', 'G', 0, 1, 2, 0xff})
	}))
	defer srv.Close()
	a := testApp(t)
	sendAndWait(t, a, srv.URL)
	msg, isErr := paneEmpty(responsePanes[0], a.cur())
	if !strings.Contains(msg, "Binary body") || !strings.Contains(msg, "8B") || isErr {
		t.Fatalf("message %q error %v", msg, isErr)
	}
	if msg, _ := paneEmpty(responsePanes[1], a.cur()); msg != "" {
		t.Fatalf("the Headers pane must still show headers, got %q", msg)
	}
}

func TestCopyBodyAndCopyAsCurl(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip

	click(a, copyBtnX, toolbarY)
	if clip.text != `{"ok":true}` {
		t.Fatalf("copy body: %q (the body as received, not the pretty one)", clip.text)
	}
	click(a, curlBtnX, toolbarY)
	if !strings.HasPrefix(clip.text, "curl -L 'http://127.0.0.1:") {
		t.Fatalf("copy as cURL: %q", clip.text)
	}
}

func TestCurlCopyExpandsVariables(t *testing.T) {
	a := testApp(t) // the sample request uses {{baseUrl}}
	a.cur().Resp = &model.Response{Status: 200, Body: []byte("{}")}
	a.cur().view.SetLines(respview.CodeLines([]byte("{}")))
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	a.step(core.Input{})
	click(a, curlBtnX, toolbarY)
	if !strings.Contains(clip.text, "https://api.github.com/search/repositories?q=gioui") || strings.Contains(clip.text, "{{") {
		t.Fatalf("cURL = %q", clip.text)
	}
}

func TestCompactChipShowsJSONCompactlyWithDeepBlocksFolded(t *testing.T) {
	a := jsonResponse(t, compactBody)
	tab := a.cur()
	pretty := len(tab.view.Visible())
	click(a, compactX, toolbarY)
	if !tab.look.Compact() || tab.look.Raw() {
		t.Fatalf("compact %v raw %v", tab.look.Compact(), tab.look.Raw())
	}
	text := viewText(tab)
	if got := len(tab.view.Visible()); got >= pretty {
		t.Fatalf("compact shows %d lines, pretty showed %d", got, pretty)
	}
	// Items are blocks at depth 2: folded, so only their previews show.
	if strings.Contains(text, `"login": "ann"`) && !strings.Contains(text, `…`) {
		t.Fatalf("deep blocks must be folded to previews:\n%s", text)
	}
	if !strings.Contains(text, `"total": 3`) {
		t.Fatalf("top-level values stay:\n%s", text)
	}
	click(a, rawX, toolbarY)
	if tab.look.Compact() || !tab.look.Raw() {
		t.Fatal("Raw replaces Compact")
	}
	click(a, prettyX, toolbarY)
	if tab.look.Compact() || tab.look.Raw() {
		t.Fatal("Pretty replaces both")
	}
}

func TestNewTabsShowResponsesCompactByDefault(t *testing.T) {
	a := testApp(t, untouchedSettings())
	if !a.cur().look.Compact() || a.cur().look.Raw() {
		t.Fatalf("compact %v raw %v", a.cur().look.Compact(), a.cur().look.Raw())
	}
	a.newRequest()
	if !a.cur().look.Compact() {
		t.Fatal("a new tab starts compact too")
	}
}

func TestResponseViewChipsAreCompactPrettyRawInThatOrder(t *testing.T) {
	a := jsonResponse(t, `{"a":1}`)
	tab := a.cur()
	click(a, rawX, toolbarY)
	if !tab.look.Raw() {
		t.Fatal("the third chip is Raw")
	}
	click(a, prettyX, toolbarY)
	if tab.look.Raw() || tab.look.Compact() {
		t.Fatal("the second chip is Pretty")
	}
	click(a, compactX, toolbarY)
	if !tab.look.Compact() {
		t.Fatal("the first chip is Compact")
	}
	if !(compactX < prettyX && prettyX < rawX) {
		t.Fatalf("order %d %d %d", compactX, prettyX, rawX)
	}
}

func TestResponseViewSettingAppliesToOpenTabsAndIsSaved(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.prefs.Search = widget.NewTextField("response view")
	clickText(t, a, "Compact", 0)
	if !a.cur().look.Compact() || a.settings.ResponseView != "compact" {
		t.Fatalf("compact %v setting %q", a.cur().look.Compact(), a.settings.ResponseView)
	}
	if saved, _ := settingsDisk.Load(path); saved.ResponseView != "compact" {
		t.Fatalf("saved %+v", saved)
	}
	clickText(t, a, "Pretty", 0)
	if a.cur().look.Compact() || a.cur().look.Raw() || a.settings.ResponseView != "pretty" {
		t.Fatalf("compact %v raw %v setting %q", a.cur().look.Compact(), a.cur().look.Raw(), a.settings.ResponseView)
	}
}
