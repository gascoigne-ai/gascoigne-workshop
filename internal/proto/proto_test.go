package proto

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestRegistryNames(t *testing.T) {
	want := []string{"mcp", "rest", "websocket"}
	if got := NewRegistry().Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestRegistrySend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	tests := []struct {
		name    string
		req     model.Request
		wantErr bool
		wantMsg string // substring of Response.Err
		want    int    // Response.Status
	}{
		{"rest", model.Request{Protocol: "rest", URL: srv.URL}, false, "", 200},
		{"websocket stub", model.Request{Protocol: "websocket"}, false, "not implemented", 0},
		{"mcp stub", model.Request{Protocol: "mcp"}, false, "not implemented", 0},
		{"unknown", model.Request{Protocol: "carrier-pigeon"}, true, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewRegistry().Send(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if resp.Status != tt.want || !strings.Contains(resp.Err, tt.wantMsg) {
				t.Fatalf("resp = %+v, want status %d err containing %q", resp, tt.want, tt.wantMsg)
			}
		})
	}
}

func TestRESTSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Echo-Method", r.Method)
		w.Header().Set("X-Echo-Auth", r.Header.Get("Authorization"))
		w.Header().Set("X-Echo-Skipped", r.Header.Get("X-Off"))
		w.WriteHeader(http.StatusCreated)
		w.Write(body)
	}))
	defer srv.Close()

	resp, err := REST{}.Send(context.Background(), model.Request{
		Method: "post",
		URL:    srv.URL,
		Body:   `{"a":1}`,
		Headers: []model.KV{
			{Name: "Authorization", Value: "Bearer t", Enabled: true},
			{Name: "X-Off", Value: "no", Enabled: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 201 || resp.StatusText != "Created" {
		t.Errorf("status = %d %q", resp.Status, resp.StatusText)
	}
	if string(resp.Body) != `{"a":1}` || resp.Size != 7 {
		t.Errorf("body = %q size %d", resp.Body, resp.Size)
	}
	if resp.Duration <= 0 {
		t.Errorf("duration = %v", resp.Duration)
	}
	got := map[string]string{}
	for _, h := range resp.Headers {
		got[h.Name] = h.Value
	}
	want := map[string]string{"X-Echo-Method": "POST", "X-Echo-Auth": "Bearer t", "X-Echo-Skipped": ""}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("header %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestRESTSendFailures(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	t.Run("bad method is an error", func(t *testing.T) {
		if _, err := (REST{}).Send(context.Background(), model.Request{Method: "BAD METHOD", URL: "http://x"}); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("connection refused lands in Err", func(t *testing.T) {
		resp, err := REST{}.Send(context.Background(), model.Request{URL: closed.URL})
		if err != nil || resp.Err == "" || resp.Status != 0 {
			t.Fatalf("resp = %+v err = %v", resp, err)
		}
	})
	t.Run("cancelled context lands in Err", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		resp, err := REST{}.Send(ctx, model.Request{URL: closed.URL})
		if err != nil || resp.Err == "" {
			t.Fatalf("resp = %+v err = %v", resp, err)
		}
	})
}
