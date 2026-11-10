package editor

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestUpdateURLTypedRebuildsParamsKeepingDescriptionsAndDisabledRows(t *testing.T) {
	req := model.Request{
		URL: "http://x/s?q=old",
		Params: []model.KV{
			{Name: "q", Value: "old", Description: "repo name", Enabled: true},
			{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
		},
	}

	got := Update(req, URLTyped{Text: "http://x/s?q=new&page=2"})

	want := []model.KV{
		{Name: "q", Value: "new", Description: "repo name", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
		{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
	}
	if got.URL != "http://x/s?q=new&page=2" || !reflect.DeepEqual(got.Params, want) {
		t.Fatalf("URL %q params %+v, want the typed URL and %+v", got.URL, got.Params, want)
	}
}

func TestUpdateParamsChangedRewritesTheQueryFromTheEnabledRows(t *testing.T) {
	req := model.Request{
		URL: "http://x/s?q=a",
		Params: []model.KV{
			{Name: "q", Value: "a", Enabled: true},
			{Name: "sort", Value: "stars", Enabled: false},
			{Name: "page", Value: "2", Enabled: true},
		},
	}

	got := Update(req, ParamsChanged{})

	if want := "http://x/s?q=a&page=2"; got.URL != want {
		t.Fatalf("URL %q, want %q", got.URL, want)
	}
}

func TestUpdateProtocolChosenFixesAMethodThatDoesNotFitRest(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		protocol   string
		wantMethod string
	}{
		{"rest with no method gets GET", "", "rest", "GET"},
		{"rest after a gRPC method gets GET", "pkg.Service/Call", "rest", "GET"},
		{"rest keeps its own method", "POST", "rest", "POST"},
		{"another protocol keeps the method", "pkg.Service/Call", "grpc", "pkg.Service/Call"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Update(model.Request{Method: c.method}, ProtocolChosen{Protocol: c.protocol})
			if got.Protocol != c.protocol || got.Method != c.wantMethod {
				t.Fatalf("protocol %q method %q, want %q and %q", got.Protocol, got.Method, c.protocol, c.wantMethod)
			}
		})
	}
}

func TestUpdateMethodChosenAndTimeoutSet(t *testing.T) {
	req := model.Request{Method: "GET", TimeoutSec: 5}

	got := Update(Update(req, MethodChosen{Method: "PUT"}), TimeoutSet{Seconds: 0})

	if got.Method != "PUT" || got.TimeoutSec != 0 {
		t.Fatalf("method %q timeout %d, want PUT and 0", got.Method, got.TimeoutSec)
	}
}

func TestUpdateDoesNotChangeTheRequestItWasGiven(t *testing.T) {
	req := model.Request{URL: "http://x?a=1", Params: []model.KV{{Name: "a", Value: "1", Enabled: true}}}
	before := req.Clone()

	Update(req, URLTyped{Text: "http://x?b=2"})
	Update(req, MethodChosen{Method: "POST"})

	if !reflect.DeepEqual(req, before) {
		t.Fatalf("the input request changed: %+v", req)
	}
}
