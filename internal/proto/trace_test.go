package proto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestRESTReportsWhereTheTimeWent(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	resp, err := REST{Client: srv.Client()}.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL})
	if err != nil || resp.Err != "" {
		t.Fatalf("%+v %v", resp, err)
	}
	tm := resp.Timing
	if tm.FirstByte < 20*time.Millisecond {
		t.Errorf("first byte %v, the server waited 20ms", tm.FirstByte)
	}
	if tm.Connect <= 0 || tm.TLS <= 0 {
		t.Errorf("a fresh TLS connection has connect %v and handshake %v", tm.Connect, tm.TLS)
	}
	if tm.Transfer < 0 || tm.FirstByte+tm.Transfer > resp.Duration+time.Millisecond {
		t.Errorf("phases %+v do not fit in %v", tm, resp.Duration)
	}
}

func TestRESTCutsLongBodiesAndSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer srv.Close()
	resp, _ := REST{MaxBody: 40}.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL})
	if !resp.Truncated || len(resp.Body) != 40 || resp.Size != 40 {
		t.Fatalf("truncated %v, %d bytes", resp.Truncated, len(resp.Body))
	}
	resp, _ = REST{MaxBody: 100}.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL})
	if resp.Truncated || len(resp.Body) != 100 {
		t.Fatalf("a body exactly at the limit is whole: truncated %v, %d bytes", resp.Truncated, len(resp.Body))
	}
}
