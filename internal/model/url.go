package model

import (
	"net/url"
	"strings"
)

// Expand replaces {{name}} with the value of the enabled variable of
// that name. Unknown names are left as written so the gap stays visible.
func Expand(s string, vars []KV) string {
	for _, v := range vars {
		if v.Enabled {
			s = strings.ReplaceAll(s, "{{"+v.Name+"}}", v.Value)
		}
	}
	return s
}

// ParamsFromURL lists the query parameters in raw, in order, all enabled.
func ParamsFromURL(raw string) []KV {
	_, query, found := strings.Cut(raw, "?")
	if !found || query == "" {
		return nil
	}
	var params []KV
	for _, pair := range strings.Split(query, "&") {
		name, value, _ := strings.Cut(pair, "=")
		params = append(params, KV{Name: unescape(name), Value: unescape(value), Enabled: true})
	}
	return params
}

// WithParams returns raw with its query replaced by the enabled params.
func WithParams(raw string, params []KV) string {
	base, _, _ := strings.Cut(raw, "?")
	var pairs []string
	for _, p := range params {
		if p.Enabled && p.Name != "" {
			pairs = append(pairs, url.QueryEscape(p.Name)+"="+url.QueryEscape(p.Value))
		}
	}
	if len(pairs) == 0 {
		return base
	}
	return base + "?" + strings.Join(pairs, "&")
}

func unescape(s string) string {
	if out, err := url.QueryUnescape(s); err == nil {
		return out
	}
	return s
}

// MergeParams rebuilds the params table after the URL was edited. The
// URL decides which enabled rows exist and their values; descriptions
// carry over from rows of the same name, and disabled rows (which the
// URL cannot hold) are kept at the end.
func MergeParams(old []KV, rawURL string) []KV {
	params := ParamsFromURL(rawURL)
	for i := range params {
		for _, o := range old {
			if o.Name == params[i].Name && o.Description != "" {
				params[i].Description = o.Description
				break
			}
		}
	}
	for _, o := range old {
		if !o.Enabled {
			params = append(params, o)
		}
	}
	return params
}
