package window

import (
	"context"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// fakePreview records what the window hands to the browser preview.
type fakePreview struct {
	shown       bool
	body        []byte
	contentType string
}

func (fake *fakePreview) Show(_ context.Context, body []byte, contentType string) error {
	fake.shown, fake.body, fake.contentType = true, body, contentType
	return nil
}

func TestOpenInBrowserHandsTheBodyOverWithItsContentType(t *testing.T) {
	a := testApp(t)
	fake := &fakePreview{}
	a.previews = fake
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("<h1>hi</h1>"),
		Headers: []request.KV{{Name: "Content-Type", Value: "text/html"}}}

	openBodyInBrowser(t, a, tab)

	if !fake.shown {
		t.Fatal("no browser was asked to open anything")
	}
	if string(fake.body) != "<h1>hi</h1>" || fake.contentType != "text/html" {
		t.Fatalf("handed over %q as %q", fake.body, fake.contentType)
	}
}

func TestOpenInBrowserFallsBackToTextPlainWithoutAContentType(t *testing.T) {
	a := testApp(t)
	fake := &fakePreview{}
	a.previews = fake
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("plain")}

	openBodyInBrowser(t, a, tab)

	if got := fake.contentType; got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}
