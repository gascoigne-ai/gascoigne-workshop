// Package preview shows a response body in the user's browser: it
// serves the bytes for a short time on a loopback port.
package preview

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
)

// secretBytes is the length of the random path that guards a page.
const secretBytes = 16

// pageCSP stops scripts, forms and plugins: the body comes from a
// server the user does not control and is shown on a local origin.
// Styles and images stay, so the page still looks like itself.
const pageCSP = "sandbox; default-src 'none'; script-src 'none'; style-src * 'unsafe-inline'; img-src * data:; font-src *; media-src *"

// Page is a body being served.
type Page struct {
	// URL is where the browser opens the body.
	URL string

	server *http.Server
	once   sync.Once
}

// Close stops the server; closing twice is fine.
func (page *Page) Close() {
	page.once.Do(func() { _ = page.server.Close() })
}

// Serve starts serving body at an unguessable path on 127.0.0.1 and
// stops after lifetime or on Close. Requests with a Host header other
// than the listener's are refused, against DNS rebinding.
func Serve(body []byte, contentType string, lifetime time.Duration) (*Page, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("preview: listen: %w", err)
	}
	secret := make([]byte, secretBytes)
	if _, err := rand.Read(secret); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("preview: random path: %w", err)
	}
	address := listener.Addr().String()
	path := "/" + hex.EncodeToString(secret)

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(writer http.ResponseWriter, request *http.Request) {
		if request.Host != address {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}
		header := writer.Header()
		header.Set("Content-Type", contentType)
		header.Set("Content-Security-Policy", pageCSP)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Cache-Control", "no-store")
		_, _ = writer.Write(body)
	})
	page := &Page{URL: "http://" + address + path, server: &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}}
	go func() { _ = page.server.Serve(listener) }()
	time.AfterFunc(lifetime, page.Close)
	return page, nil
}

// openers is the command that opens a URL in the default browser, per
// GOOS; the URL is appended.
var openers = map[string][]string{
	"linux":   {"xdg-open"},
	"darwin":  {"open"},
	"windows": {"rundll32", "url.dll,FileProtocolHandler"},
}

// openCommand is the command line that opens address, nil when the
// system is not known.
func openCommand(goos, address string) []string {
	opener, ok := openers[goos]
	if !ok {
		return nil
	}
	return append(append([]string(nil), opener...), address)
}

// OpenInBrowser asks the system to open address in the default browser.
func OpenInBrowser(address string) error {
	command := openCommand(runtime.GOOS, address)
	if command == nil {
		return fmt.Errorf("preview: no way to open a browser on %s", runtime.GOOS)
	}
	if err := exec.Command(command[0], command[1:]...).Start(); err != nil {
		return fmt.Errorf("preview: open %s: %w", command[0], err)
	}
	return nil
}

// Browser serves a body for Lifetime and opens it with Open, which is
// OpenInBrowser when nil.
type Browser struct {
	Lifetime time.Duration
	Open     func(address string) error
}

// Show implements port.BodyPreview.
func (browser Browser) Show(body []byte, contentType string) error {
	page, err := Serve(body, contentType, browser.Lifetime)
	if err != nil {
		return err
	}
	open := browser.Open
	if open == nil {
		open = OpenInBrowser
	}
	if err := open(page.URL); err != nil {
		page.Close()
		return err
	}
	return nil
}

var _ port.BodyPreview = Browser{}
