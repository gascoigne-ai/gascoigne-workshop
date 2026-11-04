package proto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	if !strings.Contains(resp.Err, "not a URL like http://host:port or socks5://host:port") {
		t.Fatalf("err %q", resp.Err)
	}
}

func writePEM(t *testing.T, name string, cert []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestACustomCAMakesAnUnknownServerTrusted(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("trusted")) }))
	defer srv.Close()
	if resp, _ := (REST{}).Send(context.Background(), model.Request{Method: "GET", URL: srv.URL}); resp.Err == "" {
		t.Fatal("without the CA the certificate is unknown")
	}
	ca := writePEM(t, "ca.pem", srv.Certificate().Raw)
	resp, _ := REST{CAFile: ca}.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL})
	if resp.Err != "" || string(resp.Body) != "trusted" {
		t.Fatalf("%+v", resp)
	}
	for file, want := range map[string]string{filepath.Join(t.TempDir(), "none.pem"): "CA file", writeFile(t, "junk.pem", "not pem"): "no PEM certificate"} {
		resp, _ := REST{CAFile: file}.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL})
		if !strings.Contains(resp.Err, want) {
			t.Errorf("CA %s: %q, want %q", file, resp.Err, want)
		}
	}
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestClientCertificateIsPresentedToServersThatAskForOne(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "client"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	certFile := writePEM(t, "client.pem", der)
	keyFile := filepath.Join(t.TempDir(), "client.key")
	os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)

	var subject string
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.PeerCertificates) > 0 {
			subject = r.TLS.PeerCertificates[0].Subject.CommonName
		}
	}))
	srv.TLS = &tls.Config{ClientAuth: tls.RequireAnyClientCert}
	srv.StartTLS()
	defer srv.Close()
	resp, _ := REST{CertFile: certFile, KeyFile: keyFile}.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL, InsecureTLS: true})
	if resp.Err != "" || subject != "client" {
		t.Fatalf("%+v subject %q", resp, subject)
	}
	resp, _ = REST{CertFile: certFile, KeyFile: filepath.Join(t.TempDir(), "none")}.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL})
	if !strings.Contains(resp.Err, "client certificate") {
		t.Fatalf("err %q", resp.Err)
	}
}

func TestProxySchemesAreChecked(t *testing.T) {
	for proxy, ok := range map[string]bool{"socks5://127.0.0.1:1080": true, "http://p:3128": true, "ftp://p:21": false, "p:3128": false} {
		_, err := REST{Proxy: proxy}.client(model.Request{})
		if (err == nil) != ok {
			t.Errorf("proxy %q: err = %v, want ok=%v", proxy, err, ok)
		}
	}
}
