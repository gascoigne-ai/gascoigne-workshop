package proto

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, strings.NewReader(req.Body))
	if err != nil {
		return model.Response{}, fmt.Errorf("build request: %w", err)
	}
	for _, h := range req.Headers {
		if h.Enabled && h.Name != "" {
			httpReq.Header.Add(h.Name, h.Value)
		}
	}

	client := r.Client
	if client == nil {
		client = &http.Client{}
	}
	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return model.Response{Err: err.Error(), Duration: time.Since(start)}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return model.Response{Err: "read body: " + err.Error(), Duration: time.Since(start)}, nil
	}
	return model.Response{
		Status:     resp.StatusCode,
		StatusText: http.StatusText(resp.StatusCode),
		Duration:   time.Since(start),
		Size:       len(body),
		Headers:    headerRows(resp.Header),
		Body:       body,
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
