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

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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
	httpReq, err := newHTTPRequest(ctx, method, req, body, contentType)
	if err != nil {
		return model.Response{}, err
	}

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

func headerRows(header http.Header) []model.KV {
	var rows []model.KV
	for name, values := range header {
		for _, v := range values {
			rows = append(rows, model.KV{Name: name, Value: v, Enabled: true})
		}
	}
	sortKV(rows)
	return rows
}

// bodyEncoders turn a request into body bytes and the Content-Type to
// send with them. BodyRaw is the legacy raw body: sent as is, with no type.
var bodyEncoders = map[model.BodyType]func(req model.Request) ([]byte, string, error){
	model.BodyRaw:  func(request model.Request) ([]byte, string, error) { return []byte(request.Body), "", nil },
	model.BodyNone: func(model.Request) ([]byte, string, error) { return nil, "", nil },
	model.BodyJSON: func(request model.Request) ([]byte, string, error) {
		return []byte(request.Body), "application/json", nil
	},
	model.BodyText: func(request model.Request) ([]byte, string, error) {
		return []byte(request.Body), "text/plain; charset=utf-8", nil
	},
	model.BodyForm:      encodeForm,
	model.BodyMultipart: encodeMultipart,
	model.BodyFile:      encodeFile,
	model.BodyGraphQL:   encodeGraphQL,
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
var authAppliers = map[model.AuthType]func(auth model.Auth, request *http.Request){
	model.AuthNone: func(model.Auth, *http.Request) {},
	model.AuthBearer: func(auth model.Auth, request *http.Request) {
		request.Header.Set("Authorization", "Bearer "+auth.Token)
	},
	model.AuthBasic: func(auth model.Auth, request *http.Request) {
		request.SetBasicAuth(auth.User, auth.Password)
	},
	model.AuthAPIKey: func(auth model.Auth, request *http.Request) {
		if auth.KeyName == "" {
			return
		}
		if auth.KeyIn == model.KeyInQuery {
			q := request.URL.Query()
			q.Set(auth.KeyName, auth.KeyValue)
			request.URL.RawQuery = q.Encode()
			return
		}
		request.Header.Set(auth.KeyName, auth.KeyValue)
	},
}

// client is the http.Client for req, built by newHTTPClient from the
// flat fields. REST keeps them flat (rather than a Network) because
// callers and tests write them that way; the other protocols take a Network.
func (r REST) client(req model.Request) (*http.Client, error) {
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
