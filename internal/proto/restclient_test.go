package proto

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestCookiesSetByOneRequestAreSentWithTheNext(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err == nil {
			got = c.Value
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
	}))
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	rest := REST{Jar: jar}
	for range 2 {
		if resp, _ := rest.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL}); resp.Err != "" {
			t.Fatal(resp.Err)
		}
	}
	if got != "abc" {
		t.Fatalf("the second request carried cookie %q", got)
	}
	if _, _ = (REST{}).Send(context.Background(), model.Request{Method: "GET", URL: srv.URL}); got != "abc" {
		t.Fatal("without a jar nothing is remembered, and nothing broke")
	}
}

func TestInsecureTLSAcceptsASelfSignedServerOnlyWhenAsked(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()
	rest := REST{} // no trusted CA: the test server's certificate is unknown
	resp, _ := rest.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL})
	if resp.Err == "" {
		t.Fatal("an unknown certificate must be refused")
	}
	resp, _ = rest.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL, InsecureTLS: true})
	if resp.Err != "" || string(resp.Body) != "ok" {
		t.Fatalf("with the switch on: %+v", resp)
	}
}

func TestProxyIsUsedAndABadOneIsExplained(t *testing.T) {
	var viaProxy atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viaProxy.Add(1)
		w.Write([]byte("from proxy for " + r.URL.Host))
	}))
	defer proxy.Close()
	resp, _ := REST{Proxy: proxy.URL}.Send(context.Background(), model.Request{Method: "GET", URL: "http://example.invalid/x"})
	if resp.Err != "" || viaProxy.Load() != 1 || !strings.Contains(string(resp.Body), "example.invalid") {
		t.Fatalf("%+v via proxy %d", resp, viaProxy.Load())
	}
	resp, _ = REST{Proxy: "not a url"}.Send(context.Background(), model.Request{Method: "GET", URL: "http://x/"})
	if !strings.Contains(resp.Err, "not a URL like http://host:port") {
		t.Fatalf("err %q", resp.Err)
	}
}
