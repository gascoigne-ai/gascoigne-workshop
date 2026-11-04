package proto

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// maxBody bounds how much of a response is kept in memory.
const maxBody = 32 << 20

// REST sends HTTP requests with net/http.
type REST struct {
	// Client is used when set; the zero value uses a plain http.Client.
	Client *http.Client

	// Jar, when set, keeps cookies between requests (shared by all of them).
	Jar http.CookieJar

	// Proxy is the URL of an HTTP proxy for every request; "" uses the
	// environment's (HTTP_PROXY and friends).
	Proxy string

	// MaxBody bounds how much of a response is kept; 0 means the default
	// of 32 MiB. A longer body is cut and the response says so.
	MaxBody int64
}

// Send implements Protocol. Request params are not applied here: the
// caller folds them into req.URL with model.WithParams.
func (r REST) Send(ctx context.Context, req model.Request) (model.Response, error) {
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = http.MethodGet
	}
	body, contentType, err := encodeBody(req)
	if err != nil {
		return model.Response{}, fmt.Errorf("encode body: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, bytes.NewReader(body))
	if err != nil {
		return model.Response{}, fmt.Errorf("build request: %w", err)
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	for _, h := range req.Headers {
		if h.Enabled && h.Name != "" {
			if strings.EqualFold(h.Name, "Content-Type") {
				httpReq.Header.Del("Content-Type") // the user's header replaces the automatic one
			}
			httpReq.Header.Add(h.Name, h.Value)
		}
	}

	apply, ok := authAppliers[req.Auth.Type]
	if !ok {
		return model.Response{}, fmt.Errorf("unknown auth type %q", req.Auth.Type)
	}
	apply(req.Auth, httpReq) // after the headers, so the Auth pane wins over one typed by hand

	client, err := r.client(req)
	if err != nil {
		return model.Response{Err: err.Error()}, nil
	}
	tr := &tracer{}
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), tr.trace()))
	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return model.Response{Err: failureText(err), Duration: time.Since(start)}, nil
	}
	defer resp.Body.Close()
	headersAt := time.Now()

	limit := r.MaxBody
	if limit <= 0 {
		limit = maxBody
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return model.Response{Err: failureText(err), Duration: time.Since(start)}, nil
	}
	truncated := int64(len(respBody)) > limit
	if truncated {
		respBody = respBody[:limit]
	}
	return model.Response{
		Timing:     tr.timing(start, headersAt, time.Now()),
		Truncated:  truncated,
		Status:     resp.StatusCode,
		StatusText: http.StatusText(resp.StatusCode),
		Duration:   time.Since(start),
		Size:       len(respBody),
		Headers:    headerRows(resp.Header),
		Body:       respBody,
	}, nil
}

func headerRows(h http.Header) []model.KV {
	var rows []model.KV
	for name, values := range h {
		for _, v := range values {
			rows = append(rows, model.KV{Name: name, Value: v, Enabled: true})
		}
	}
	sortKV(rows)
	return rows
}

// bodyEncoders turn a request into body bytes and the Content-Type to
// send with them. "" is the legacy raw body: sent as is, with no type.
var bodyEncoders = map[string]func(req model.Request) ([]byte, string, error){
	"":                  func(r model.Request) ([]byte, string, error) { return []byte(r.Body), "", nil },
	model.BodyNone:      func(model.Request) ([]byte, string, error) { return nil, "", nil },
	model.BodyJSON:      func(r model.Request) ([]byte, string, error) { return []byte(r.Body), "application/json", nil },
	model.BodyText:      func(r model.Request) ([]byte, string, error) { return []byte(r.Body), "text/plain; charset=utf-8", nil },
	model.BodyForm:      encodeForm,
	model.BodyMultipart: encodeMultipart,
	model.BodyFile:      encodeFile,
}

func encodeBody(req model.Request) ([]byte, string, error) {
	enc, ok := bodyEncoders[req.BodyType]
	if !ok {
		return nil, "", fmt.Errorf("unknown body type %q", req.BodyType)
	}
	return enc(req)
}

// formFields are the enabled, named rows of the form.
func formFields(req model.Request) []model.KV {
	var fields []model.KV
	for _, f := range req.Form {
		if f.Enabled && f.Name != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func encodeForm(req model.Request) ([]byte, string, error) {
	var pairs []string
	for _, f := range formFields(req) {
		pairs = append(pairs, url.QueryEscape(f.Name)+"="+url.QueryEscape(f.Value))
	}
	return []byte(strings.Join(pairs, "&")), "application/x-www-form-urlencoded", nil
}

func encodeMultipart(req model.Request) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range formFields(req) {
		if path, ok := strings.CutPrefix(f.Value, "@"); ok && path != "" && !strings.HasPrefix(path, "@") {
			if err := writeFilePart(w, f.Name, path); err != nil {
				return nil, "", err
			}
			continue
		}
		value := strings.TrimPrefix(f.Value, "@") // "@@x" sends the text "@x"
		if !strings.HasPrefix(f.Value, "@@") {
			value = f.Value
		}
		if err := w.WriteField(f.Name, value); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

// failureText is the message shown for a failed request: a stopped one
// says so plainly, anything else keeps the transport's own words.
func failureText(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return ErrCancelled
	case errors.Is(err, context.DeadlineExceeded):
		return ErrTimedOut
	}
	return err.Error()
}

// authAppliers add credentials to a request, by auth type. The zero
// value of the type is no authentication.
var authAppliers = map[string]func(a model.Auth, r *http.Request){
	"":             func(model.Auth, *http.Request) {},
	model.AuthNone: func(model.Auth, *http.Request) {},
	model.AuthBearer: func(a model.Auth, r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+a.Token)
	},
	model.AuthBasic: func(a model.Auth, r *http.Request) {
		r.SetBasicAuth(a.User, a.Password)
	},
	model.AuthAPIKey: func(a model.Auth, r *http.Request) {
		if a.KeyName == "" {
			return
		}
		if a.KeyIn == model.KeyInQuery {
			q := r.URL.Query()
			q.Set(a.KeyName, a.KeyValue)
			r.URL.RawQuery = q.Encode()
			return
		}
		r.Header.Set(a.KeyName, a.KeyValue)
	},
}

// client is the http.Client for req: the configured one, with the cookie
// jar, the proxy, the certificate check and the redirect rule of the
// request applied to a copy.
func (r REST) client(req model.Request) (*http.Client, error) {
	c := &http.Client{}
	if r.Client != nil {
		cp := *r.Client
		c = &cp
	}
	if c.Jar == nil {
		c.Jar = r.Jar
	}
	if req.NoRedirects {
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	if r.Proxy == "" && !req.InsecureTLS {
		return c, nil
	}
	var base *http.Transport
	switch t := c.Transport.(type) {
	case nil:
		base = http.DefaultTransport.(*http.Transport).Clone()
	case *http.Transport:
		base = t.Clone()
	default:
		return c, nil // a custom round tripper is left alone
	}
	if r.Proxy != "" {
		u, err := url.Parse(r.Proxy)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("the proxy %q is not a URL like http://host:port", r.Proxy)
		}
		base.Proxy = http.ProxyURL(u)
	}
	if req.InsecureTLS {
		if base.TLSClientConfig == nil {
			base.TLSClientConfig = &tls.Config{}
		}
		base.TLSClientConfig.InsecureSkipVerify = true //nolint: the request asked for it
	}
	c.Transport = base
	return c, nil
}

// fileContentType is the Content-Type of a file by its extension.
func fileContentType(path string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// writeFilePart adds the file at path as a multipart part called name.
func writeFilePart(w *multipart.Writer, name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("form field %q: %w", name, err)
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, name, filepath.Base(path)))
	h.Set("Content-Type", fileContentType(path))
	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

// encodeFile sends the file named by the body as it is.
func encodeFile(req model.Request) ([]byte, string, error) {
	path := strings.TrimSpace(req.Body)
	if path == "" {
		return nil, "", fmt.Errorf("choose the file to send")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("body file: %w", err)
	}
	return data, fileContentType(path), nil
}
