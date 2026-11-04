package respview

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestSuggestedFileNameFollowsTheContentType(t *testing.T) {
	for ct, want := range map[string]string{
		"application/json; charset=utf-8": "response.json",
		"text/html":                       "response.html",
		"application/vnd.api+json":        "response.json",
		"image/png":                       "response.png",
		"":                                "response.txt",
	} {
		r := &model.Response{}
		if ct != "" {
			r.Headers = []model.KV{{Name: "content-type", Value: ct, Enabled: true}}
		}
		if got := SuggestedFileName(r); got != want {
			t.Errorf("%q: %q, want %q", ct, got, want)
		}
	}
}
