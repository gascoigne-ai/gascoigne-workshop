package model

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestExpand(t *testing.T) {
	vars := []KV{
		{Name: "baseUrl", Value: "https://api.github.com", Enabled: true},
		{Name: "off", Value: "x", Enabled: false},
	}
	tests := []struct{ name, in, want string }{
		{"known", "{{baseUrl}}/search", "https://api.github.com/search"},
		{"unknown stays", "{{nope}}/a", "{{nope}}/a"},
		{"disabled stays", "{{off}}", "{{off}}"},
		{"none", "plain", "plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Expand(tt.in, vars); got != tt.want {
				t.Fatalf("Expand(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParamsFromURL(t *testing.T) {
	tests := []struct {
		name, in string
		want     []KV
	}{
		{"repo search", "{{baseUrl}}/search/repositories?q=gioui&order=desc&sort=stars",
			[]KV{{Name: "q", Value: "gioui", Enabled: true}, {Name: "order", Value: "desc", Enabled: true}, {Name: "sort", Value: "stars", Enabled: true}}},
		{"no query", "http://x/y", nil},
		{"empty query", "http://x/y?", nil},
		{"flag without value", "http://x?debug", []KV{{Name: "debug", Enabled: true}}},
		{"escaped", "http://x?q=a%20b", []KV{{Name: "q", Value: "a b", Enabled: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParamsFromURL(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestWithParams(t *testing.T) {
	tests := []struct {
		name, in string
		params   []KV
		want     string
	}{
		{"replaces query", "http://x/y?old=1", []KV{{Name: "q", Value: "a b", Enabled: true}}, "http://x/y?q=a+b"},
		{"skips disabled and unnamed", "http://x/y", []KV{{Name: "a", Value: "1"}, {Value: "2", Enabled: true}}, "http://x/y"},
		{"keeps template base", "{{baseUrl}}/s?z=1", []KV{{Name: "q", Value: "1", Enabled: true}}, "{{baseUrl}}/s?q=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithParams(tt.in, tt.params); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeParams(t *testing.T) {
	old := []KV{
		{Name: "q", Value: "old", Description: "repo name", Enabled: true},
		{Name: "order", Value: "desc", Enabled: true},
		{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
	}
	got := MergeParams(old, "http://x/s?q=new&page=2")
	want := []KV{
		{Name: "q", Value: "new", Description: "repo name", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
		{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

// A query with a broken escape must keep its text instead of dropping it.
func TestParamsFromURLKeepsBrokenEscapes(t *testing.T) {
	got := ParamsFromURL("http://x?q=%zz&%GG=1")
	want := []KV{{Name: "q", Value: "%zz", Enabled: true}, {Name: "%GG", Value: "1", Enabled: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCloneIsIndependentAndNormalizesEmptySlices(t *testing.T) {
	orig := Request{
		Name: "r", Params: []KV{{Name: "a", Value: "1"}}, Headers: []KV{}, Form: nil,
	}
	c := orig.Clone()
	c.Params[0].Value = "changed"
	c.Params = append(c.Params, KV{Name: "b"})
	if orig.Params[0].Value != "1" || len(orig.Params) != 1 {
		t.Fatalf("editing the clone changed the original: %+v", orig.Params)
	}
	if c.Headers != nil || c.Form != nil {
		t.Fatalf("empty slices must clone to nil: %#v %#v", c.Headers, c.Form)
	}
	if !reflect.DeepEqual(orig.Clone(), (&Request{Name: "r", Params: []KV{{Name: "a", Value: "1"}}}).Clone()) {
		t.Fatal("equal requests must have equal clones, whatever their empty slices were")
	}
}

func TestRequestExpandedReplacesVariablesEverywhereWithoutTouchingTheOriginal(t *testing.T) {
	vars := []KV{{Name: "host", Value: "api.test", Enabled: true}, {Name: "tok", Value: "s3cret", Enabled: true}}
	orig := Request{
		URL:     "https://{{host}}/x",
		Headers: []KV{{Name: "X-{{host}}", Value: "{{tok}}", Enabled: true}},
		Body:    `{"h":"{{host}}"}`,
		Form:    []KV{{Name: "{{host}}", Value: "{{tok}}", Enabled: true}},
		Auth: Auth{Type: AuthBasic, Token: "{{tok}}", User: "{{host}}", Password: "{{tok}}",
			KeyName: "{{host}}", KeyValue: "{{tok}}"},
	}
	got := orig.Expanded(vars)

	if got.URL != "https://api.test/x" || got.Body != `{"h":"api.test"}` {
		t.Errorf("url %q body %q", got.URL, got.Body)
	}
	if got.Headers[0].Name != "X-api.test" || got.Headers[0].Value != "s3cret" {
		t.Errorf("headers %+v", got.Headers)
	}
	if got.Form[0].Name != "api.test" || got.Form[0].Value != "s3cret" {
		t.Errorf("form %+v", got.Form)
	}
	a := got.Auth
	if a.Token != "s3cret" || a.User != "api.test" || a.Password != "s3cret" || a.KeyName != "api.test" || a.KeyValue != "s3cret" {
		t.Errorf("auth %+v", a)
	}
	if orig.URL != "https://{{host}}/x" || orig.Headers[0].Value != "{{tok}}" || orig.Form[0].Value != "{{tok}}" || orig.Auth.Token != "{{tok}}" {
		t.Fatal("Expanded must not change the original request")
	}
}

func TestRequestExpandedLeavesUnknownVariablesVisible(t *testing.T) {
	got := Request{URL: "{{nope}}/a", Auth: Auth{Token: "{{nope}}"}}.Expanded(nil)
	if got.URL != "{{nope}}/a" || got.Auth.Token != "{{nope}}" {
		t.Fatalf("%+v", got)
	}
}

func TestPathParametersFollowTheURLAndFillIt(t *testing.T) {
	names := PathParamNames("https://api.test/users/:id/posts/:post?x=:notme#:nor")
	if len(names) != 2 || names[0] != "id" || names[1] != "post" {
		t.Fatalf("names %v", names)
	}
	if got := PathParamNames("https://:8080/{{x}}/:{{y}}/:ok"); len(got) != 1 || got[0] != "ok" {
		t.Fatalf("a port, a variable and a mixed segment are not path parameters: %v", got)
	}
	rows := MergeParams([]KV{{Name: ":id", Value: "42", Description: "user", Enabled: true}, {Name: ":gone", Value: "x", Enabled: true}}, "https://api.test/users/:id/posts/:post?q=1")
	if len(rows) != 3 || rows[0].Name != "q" || rows[1].Name != ":id" || rows[1].Value != "42" || rows[1].Description != "user" || rows[2].Name != ":post" || rows[2].Value != "" {
		t.Fatalf("rows %+v", rows)
	}
	if got := WithParams("https://api.test/users/:id", rows); got != "https://api.test/users/:id?q=1" {
		t.Fatalf("path rows must not go into the query: %q", got)
	}
	r := Request{URL: "https://api.test/users/:id/posts/:post?q=1", Params: rows}
	r.Params[1].Value = "a b/c"
	if got := r.Expanded(nil).URL; got != "https://api.test/users/a%20b%2Fc/posts/:post?q=1" {
		t.Fatalf("expanded %q", got)
	}
	r.Params[1].Enabled = false
	if got := r.Expanded(nil).URL; !strings.Contains(got, "/users/:id/") {
		t.Fatalf("a disabled row leaves the segment: %q", got)
	}
	vars := []KV{{Name: "uid", Value: "7", Enabled: true}}
	r2 := Request{URL: "http://h/u/:id", Params: []KV{{Name: ":id", Value: "{{uid}}", Enabled: true}}}
	if got := r2.Expanded(vars).URL; got != "http://h/u/7" {
		t.Fatalf("a path value may use variables: %q", got)
	}
}

// Variables that each hold two of the next ("a={{b}}{{b}}", "b={{c}}{{c}}"
// ...) doubled the text at every step, so 30 of them, from an imported
// collection, ate all memory. A replacement that would pass the limit is
// skipped and its name stays visible.
func TestExpandStopsAtASizeLimit(t *testing.T) {
	var vars []KV
	for i := range 40 {
		vars = append(vars, KV{Name: fmt.Sprintf("v%d", i), Value: fmt.Sprintf("{{v%d}}{{v%d}}", i+1, i+1), Enabled: true})
	}
	got := Expand("{{v0}}", vars)
	if len(got) > maxExpanded {
		t.Fatalf("expanded to %d bytes, the limit is %d", len(got), maxExpanded)
	}
	if !strings.Contains(got, "{{v") {
		t.Errorf("a skipped variable must stay visible, got %.40q", got)
	}
}

// FuzzURLParams: query and path parameters of any URL text.
func FuzzURLParams(fuzz *testing.F) {
	for _, seed := range []string{
		"https://api.test/s?q=go+lang&page=2", "https://api.test/u/:id/posts/{postId}?x=%zz", "http://[::1]:80/a?b", "?", "{{base}}/p?a=1&a=2", "",
	} {
		fuzz.Add(seed)
	}
	fuzz.Fuzz(func(t *testing.T, raw string) {
		params := ParamsFromURL(raw)
		_ = WithParams(raw, params)
		_ = ApplyPathParams(raw, params)
		_ = PathParamNames(raw)
		_ = VarNames(raw)
		_ = Expand(raw, []KV{{Name: "base", Value: raw, Enabled: true}})
	})
}

// A value captured from a response ("set x = json x") was expanded again
// by the variables after it, so a server reply holding {{secret}} pulled
// the secret into the next request. A captured (Literal) value is text.
func TestExpandNeverExpandsInsideACapturedValue(t *testing.T) {
	vars := []KV{
		{Name: "captured", Value: "{{secret}}", Enabled: true, Literal: true},
		{Name: "secret", Value: "hunter2", Enabled: true},
		{Name: "base", Value: "{{host}}/api", Enabled: true},
		{Name: "host", Value: "x.test", Enabled: true},
	}
	if got := Expand("{{captured}} {{secret}} {{base}}", vars); got != "{{secret}} hunter2 x.test/api" {
		t.Fatalf("got %q: a captured value stays literal, ordinary variables still chain", got)
	}
	if got := Expand("{{captured}}", UpsertVar(nil, "id", "{{secret}}")); got != "{{captured}}" {
		t.Fatalf("got %q for a variable that does not exist", got)
	}
	rows := UpsertVar(nil, "id", "{{secret}}")
	if !rows[0].Literal {
		t.Error("UpsertVar (the capture path) must mark the row Literal")
	}
}
