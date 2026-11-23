package protocol

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Network is the connection setting every HTTP-based protocol shares. The
// zero value is a plain client.
type Network struct {
	// Jar, when set, keeps cookies between requests.
	Jar http.CookieJar

	// Proxy is the URL of a proxy, http://host:port or socks5://host:port.
	Proxy string

	// CAFile is a PEM file of authorities to trust in addition to the
	// system's; CertFile and KeyFile are the client certificate.
	CAFile, CertFile, KeyFile string
}

// proxySchemes are the proxy URL schemes net/http can dial.
var proxySchemes = []string{"http", "https", "socks5", "socks5h"}

// checkAuth reports an auth type no applier knows. It is its own step so a
// protocol that connects before it builds a request (MCP) fails early too.
func checkAuth(auth request.Auth) error {
	if _, ok := authAppliers[auth.Type]; !ok {
		return fmt.Errorf("unknown auth type %q", auth.Type)
	}
	return nil
}

// newHTTPRequest builds the request every HTTP-based protocol sends: the
// body with its automatic Content-Type, then every enabled named header
// (same names are all sent; a typed Content-Type replaces the automatic
// one), then the credentials of the Auth pane last, so they win over a
// header typed by hand. Protocols set their own headers afterwards.
func newHTTPRequest(ctx context.Context, method string, call request.Request, body []byte, contentType string) (*http.Request, error) {
	if err := checkAuth(call.Auth); err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, call.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	replaced := false
	for _, header := range call.Headers {
		if !header.Enabled || header.Name == "" {
			continue
		}
		if !replaced && strings.EqualFold(header.Name, "Content-Type") {
			httpReq.Header.Del("Content-Type")
			replaced = true
		}
		httpReq.Header.Add(header.Name, header.Value)
	}
	authAppliers[call.Auth.Type](call.Auth, httpReq)
	return httpReq, nil
}

// newHTTPClient is the http.Client for request: a copy of base (a plain
// client when nil) with the cookie jar, the proxy, the certificate check
// and the redirect rule applied. A custom round tripper is left alone.
func newHTTPClient(base *http.Client, network Network, call request.Request) (*http.Client, error) {
	client := &http.Client{}
	if base != nil {
		copied := *base
		client = &copied
	}
	if client.Jar == nil {
		client.Jar = network.Jar
	}
	if call.NoRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	if network.Proxy == "" && !call.InsecureTLS && network.CAFile == "" && network.CertFile == "" {
		return client, nil
	}
	var transport *http.Transport
	switch current := client.Transport.(type) {
	case nil:
		transport = http.DefaultTransport.(*http.Transport).Clone()
	case *http.Transport:
		transport = current.Clone()
	default:
		return client, nil
	}
	if network.Proxy != "" {
		proxy, err := url.Parse(network.Proxy)
		if err != nil || proxy.Host == "" || !slices.Contains(proxySchemes, proxy.Scheme) {
			return nil, fmt.Errorf("the proxy %q is not a URL like http://host:port or socks5://host:port", network.Proxy)
		}
		transport.Proxy = http.ProxyURL(proxy)
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	if err := applyTLS(transport.TLSClientConfig, network, call); err != nil {
		return nil, err
	}
	client.Transport = transport
	return client, nil
}

// applyTLS adds the configured authorities and client certificate to cfg,
// and turns the certificate check off when the request asks for it.
func applyTLS(cfg *tls.Config, network Network, call request.Request) error {
	if network.CAFile != "" {
		pem, err := os.ReadFile(network.CAFile)
		if err != nil {
			return fmt.Errorf("CA file: %w", err)
		}
		pool := cfg.RootCAs
		if pool == nil {
			if pool, _ = x509.SystemCertPool(); pool == nil {
				pool = x509.NewCertPool()
			}
		}
		if !pool.AppendCertsFromPEM(pem) {
			return fmt.Errorf("CA file %s holds no PEM certificate", filepath.Base(network.CAFile))
		}
		cfg.RootCAs = pool
	}
	if network.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(network.CertFile, network.KeyFile)
		if err != nil {
			return fmt.Errorf("client certificate: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	if call.InsecureTLS {
		cfg.InsecureSkipVerify = true //nolint: the request asked for it
	}
	return nil
}
