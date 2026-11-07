package preview

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func get(t *testing.T, address, host string) (*http.Response, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		request.Host = host
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response, string(body)
}

func TestServeAnswersWithTheBodyAndItsContentType(t *testing.T) {
	page, err := Serve([]byte("<h1>hi</h1>"), "text/html; charset=utf-8", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	response, body := get(t, page.URL, "")
	if response.StatusCode != 200 || body != "<h1>hi</h1>" {
		t.Fatalf("status %d body %q", response.StatusCode, body)
	}
	if got := response.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
}

// The body comes from a server the user does not control, and the page
// lives on a local origin: scripts must not run, or they could reach
// other services on this machine.
func TestServedPagesCannotRunScripts(t *testing.T) {
	page, err := Serve([]byte("<script>1</script>"), "text/html", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	response, _ := get(t, page.URL, "")
	csp := response.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "sandbox") || !strings.Contains(csp, "script-src 'none'") {
		t.Errorf("Content-Security-Policy = %q", csp)
	}
	if response.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("the browser must not guess another content type")
	}
}

func TestServeListensOnTheLoopbackOnly(t *testing.T) {
	page, err := Serve([]byte("x"), "text/plain", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	parsed, err := url.Parse(page.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, _, _ := net.SplitHostPort(parsed.Host)
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		t.Fatalf("URL host %q is not a loopback address", host)
	}
}

func TestOnlyTheSecretPathAnswers(t *testing.T) {
	page, err := Serve([]byte("x"), "text/plain", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	root := page.URL[:strings.LastIndex(page.URL, "/")+1]
	if response, _ := get(t, root, ""); response.StatusCode != http.StatusNotFound {
		t.Errorf("the root answers %d", response.StatusCode)
	}
	if response, _ := get(t, root+"guess", ""); response.StatusCode != http.StatusNotFound {
		t.Errorf("another path answers %d", response.StatusCode)
	}
}

// A page on another origin can point its own name at 127.0.0.1 (DNS
// rebinding) to read the preview; a foreign Host header gives it away.
func TestAForeignHostHeaderIsRefused(t *testing.T) {
	page, err := Serve([]byte("x"), "text/plain", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	if response, _ := get(t, page.URL, "evil.example"); response.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", response.StatusCode)
	}
}

func TestServeStopsAfterItsLifetime(t *testing.T) {
	page, err := Serve([]byte("x"), "text/plain", 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := http.Get(page.URL); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the server still answers after its lifetime")
}

func TestCloseStopsTheServer(t *testing.T) {
	page, err := Serve([]byte("x"), "text/plain", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	page.Close()
	page.Close() // closing twice is fine
	if _, err := http.Get(page.URL); err == nil {
		t.Fatal("the server answers after Close")
	}
}

func TestOpenCommandPerSystem(t *testing.T) {
	tests := []struct {
		goos string
		want []string
	}{
		{"linux", []string{"xdg-open", "http://x"}},
		{"darwin", []string{"open", "http://x"}},
		{"windows", []string{"rundll32", "url.dll,FileProtocolHandler", "http://x"}},
		{"plan9", nil},
	}
	for _, tt := range tests {
		got := openCommand(tt.goos, "http://x")
		if strings.Join(got, " ") != strings.Join(tt.want, " ") {
			t.Errorf("openCommand(%q) = %v, want %v", tt.goos, got, tt.want)
		}
	}
}
