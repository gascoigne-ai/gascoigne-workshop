package respview

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestSuggestedFileNameFollowsTheContentType(t *testing.T) {
	for ct, want := range map[string]string{
		"application/json; charset=utf-8": "response.json",
		"text/html":                       "response.html",
		"application/vnd.api+json":        "response.json",
		"image/png":                       "response.png",
		"":                                "response.txt",
	} {
		r := &request.Response{}
		if ct != "" {
			r.Headers = []request.KV{{Name: "content-type", Value: ct, Enabled: true}}
		}
		if got := SuggestedFileName(r); got != want {
			t.Errorf("%q: %q, want %q", ct, got, want)
		}
	}
}

func TestContentTypeIsTheHeaderValueAsReceived(t *testing.T) {
	tests := []struct {
		name    string
		headers []request.KV
		want    string
	}{
		{"with a charset", []request.KV{{Name: "content-type", Value: "text/html; charset=utf-8"}}, "text/html; charset=utf-8"},
		{"no header", []request.KV{{Name: "X-A", Value: "b"}}, ""},
	}
	for _, tt := range tests {
		if got := ContentType(&request.Response{Headers: tt.headers}); got != tt.want {
			t.Errorf("%s: ContentType = %q, want %q", tt.name, got, tt.want)
		}
	}
}
