package proto

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
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

	client := r.Client
	if client == nil {
		client = &http.Client{}
	}
	if req.NoRedirects {
		stop := *client
		stop.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		client = &stop
	}
	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return model.Response{Err: failureText(err), Duration: time.Since(start)}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return model.Response{Err: failureText(err), Duration: time.Since(start)}, nil
	}
	return model.Response{
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
		if err := w.WriteField(f.Name, f.Value); err != nil {
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
