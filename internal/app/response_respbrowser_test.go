package app

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/adapter/preview"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func TestOpenInBrowserServesTheBodyAndOpensItsAddress(t *testing.T) {
	a := testApp(t)
	var opened string
	a.previews = preview.Browser{Lifetime: time.Minute, Open: func(_ context.Context, address string) error { opened = address; return nil }}
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("<h1>hi</h1>"),
		Headers: []request.KV{{Name: "Content-Type", Value: "text/html"}}}

	a.openInBrowser(tab)

	if opened == "" {
		t.Fatal("no browser was asked to open anything")
	}
	response, err := http.Get(opened)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if string(body) != "<h1>hi</h1>" || response.Header.Get("Content-Type") != "text/html" {
		t.Fatalf("served %q as %q", body, response.Header.Get("Content-Type"))
	}
}

func TestOpenInBrowserWithoutAResponseOpensNothing(t *testing.T) {
	a := testApp(t)
	a.previews = preview.Browser{Lifetime: time.Minute, Open: func(context.Context, string) error { t.Fatal("a browser was opened"); return nil }}
	a.openInBrowser(a.cur())
}

func TestOpenInBrowserFallsBackToTextPlainWithoutAContentType(t *testing.T) {
	a := testApp(t)
	var opened string
	a.previews = preview.Browser{Lifetime: time.Minute, Open: func(_ context.Context, address string) error { opened = address; return nil }}
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("plain")}
	a.openInBrowser(tab)
	response, err := http.Get(opened)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if got := response.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}
