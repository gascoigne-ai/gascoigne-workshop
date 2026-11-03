package model

import (
	"reflect"
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
