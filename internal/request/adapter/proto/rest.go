package proto

import (
	"bytes"
	"context"
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

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// maxBody bounds how much of a response is kept in memory.
const maxBody = 32 << 20

// REST sends HTTP requests with net/http.
type REST struct {
	// Client is used when set; the zero value uses a plain http.Client.
	Client *http.Client

	// Jar, when set, keeps cookies between requests (shared by all of them).
	Jar http.CookieJar

	// Proxy is the URL of a proxy for every request, http://host:port,
	// https://... or socks5://host:port; "" uses the environment's
	// (HTTP_PROXY and friends).
	Proxy string

	// CAFile is a PEM file of certificate authorities to trust in addition
	// to the system's; CertFile and KeyFile are a client certificate and its
	// key for servers that ask for one.
	CAFile, CertFile, KeyFile string

	// MaxBody bounds how much of a response is kept; 0 means the default
	// of 32 MiB. A longer body is cut and the response says so.
	MaxBody int64
}

// Send implements Protocol. Request params are not applied here: the
// caller folds them into req.URL with request.WithParams.
func (r REST) Send(ctx context.Context, req request.Request) (request.Response, error) {
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = http.MethodGet
	}
	body, contentType, err := encodeBody(req)
	if err != nil {
		return request.Response{}, fmt.Errorf("encode body: %w", err)
	}
	httpReq, err := newHTTPRequest(ctx, method, req, body, contentType)
	if err != nil {
		return request.Response{}, err
	}

	client, err := r.client(req)
	if err != nil {
		return request.Response{Err: err.Error()}, nil
	}
	tr := &tracer{}
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), tr.trace()))
	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return request.Response{Err: failureText(err), Duration: time.Since(start)}, nil
	}
	defer resp.Body.Close()
	headersAt := time.Now()

	limit := r.MaxBody
	if limit <= 0 {
		limit = maxBody
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return request.Response{Err: failureText(err), Duration: time.Since(start)}, nil
	}
	truncated := int64(len(respBody)) > limit
	if truncated {
		respBody = respBody[:limit]
	}
	return request.Response{
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

func headerRows(header http.Header) []request.KV {
	var rows []request.KV
	for name, values := range header {
		for _, v := range values {
			rows = append(rows, request.KV{Name: name, Value: v, Enabled: true})
		}
	}
	sortKV(rows)
	return rows
}

// bodyEncoders turn a request into body bytes and the Content-Type to
// send with them. BodyRaw is the legacy raw body: sent as is, with no type.
var bodyEncoders = map[request.BodyType]func(req request.Request) ([]byte, string, error){
	request.BodyRaw:  func(call request.Request) ([]byte, string, error) { return []byte(call.Body), "", nil },
	request.BodyNone: func(request.Request) ([]byte, string, error) { return nil, "", nil },
	request.BodyJSON: func(call request.Request) ([]byte, string, error) {
		return []byte(call.Body), "application/json", nil
	},
	request.BodyText: func(call request.Request) ([]byte, string, error) {
		return []byte(call.Body), "text/plain; charset=utf-8", nil
	},
	request.BodyForm:      encodeForm,
	request.BodyMultipart: encodeMultipart,
	request.BodyFile:      encodeFile,
	request.BodyGraphQL:   encodeGraphQL,
}

func encodeBody(req request.Request) ([]byte, string, error) {
	enc, ok := bodyEncoders[req.BodyType]
	if !ok {
		return nil, "", fmt.Errorf("unknown body type %q", req.BodyType)
	}
	return enc(req)
}

// formFields are the enabled, named rows of the form.
func formFields(req request.Request) []request.KV {
	var fields []request.KV
	for _, f := range req.Form {
		if f.Enabled && f.Name != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func encodeForm(req request.Request) ([]byte, string, error) {
	var pairs []string
	for _, f := range formFields(req) {
		pairs = append(pairs, url.QueryEscape(f.Name)+"="+url.QueryEscape(f.Value))
	}
	return []byte(strings.Join(pairs, "&")), "application/x-www-form-urlencoded", nil
}

func encodeMultipart(req request.Request) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range formFields(req) {
		if path, ok := request.FormFile(f.Value); ok {
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
		return request.ErrCancelled
	case errors.Is(err, context.DeadlineExceeded):
		return request.ErrTimedOut
	}
	return err.Error()
}

// authAppliers add credentials to a request, by auth type. The zero
// value of the type is no authentication.
var authAppliers = map[request.AuthType]func(auth request.Auth, call *http.Request){
	request.AuthNone: func(request.Auth, *http.Request) {},
	request.AuthBearer: func(auth request.Auth, call *http.Request) {
		call.Header.Set("Authorization", "Bearer "+auth.Token)
	},
	request.AuthBasic: func(auth request.Auth, call *http.Request) {
		call.SetBasicAuth(auth.User, auth.Password)
	},
	request.AuthAPIKey: func(auth request.Auth, call *http.Request) {
		if auth.KeyName == "" {
			return
		}
		if auth.KeyIn == request.KeyInQuery {
			q := call.URL.Query()
			q.Set(auth.KeyName, auth.KeyValue)
			call.URL.RawQuery = q.Encode()
			return
		}
		call.Header.Set(auth.KeyName, auth.KeyValue)
	},
}

// client is the http.Client for req, built by newHTTPClient from the
// flat fields. REST keeps them flat (rather than a Network) because
// callers and tests write them that way; the other protocols take a Network.
func (r REST) client(req request.Request) (*http.Client, error) {
	return newHTTPClient(r.Client, Network{Jar: r.Jar, Proxy: r.Proxy, CAFile: r.CAFile, CertFile: r.CertFile, KeyFile: r.KeyFile}, req)
}

// fileContentType is the Content-Type of a file by its extension.
func fileContentType(path string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// writeFilePart adds the file at path as a multipart part called name.
func writeFilePart(form *multipart.Writer, name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("form field %q: %w", name, err)
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, name, filepath.Base(path)))
	h.Set("Content-Type", fileContentType(path))
	part, err := form.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

// encodeFile sends the file named by the body as it is.
func encodeFile(req request.Request) ([]byte, string, error) {
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
