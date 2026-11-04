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
		if p.Enabled && p.Name != "" && !IsPathParam(p.Name) {
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
		if !o.Enabled && !IsPathParam(o.Name) {
			params = append(params, o)
		}
	}
	// path parameters follow the URL: one row for each :name segment, with
	// the value and description of the row it had
	for _, name := range PathParamNames(rawURL) {
		row := KV{Name: ":" + name, Enabled: true}
		for _, o := range old {
			if o.Name == row.Name {
				row.Value, row.Description, row.Enabled = o.Value, o.Description, o.Enabled
				break
			}
		}
		params = append(params, row)
	}
	return params
}

// IsPathParam reports whether a Params row name stands for a :name segment
// of the URL path (":id") and not for a query parameter.
func IsPathParam(name string) bool { return len(name) > 1 && name[0] == ':' }

// pathSpan is where the path of raw lies: after the host of a URL with a
// scheme, or the start for a bare path; up to the query or fragment.
func pathSpan(raw string) (start, end int) {
	start = 0
	if i := strings.Index(raw, "://"); i >= 0 {
		if j := strings.IndexByte(raw[i+3:], '/'); j >= 0 {
			start = i + 3 + j
		} else {
			start = len(raw)
		}
	}
	end = len(raw)
	if k := strings.IndexAny(raw[start:], "?#"); k >= 0 {
		end = start + k
	}
	return start, end
}

// PathParamNames lists the :name segments of the URL path, in order and
// without repeats. {{variables}} are not path parameters.
func PathParamNames(raw string) []string {
	start, end := pathSpan(raw)
	var names []string
	seen := map[string]bool{}
	for _, seg := range strings.Split(raw[start:end], "/") {
		if IsPathParam(seg) && !strings.Contains(seg, "{{") && !seen[seg] {
			seen[seg] = true
			names = append(names, seg[1:])
		}
	}
	return names
}

// ApplyPathParams replaces each :name segment of the URL path with the
// escaped value of the enabled ":name" row; segments without a value stay.
func ApplyPathParams(raw string, params []KV) string {
	start, end := pathSpan(raw)
	segs := strings.Split(raw[start:end], "/")
	for i, seg := range segs {
		if !IsPathParam(seg) {
			continue
		}
		for _, p := range params {
			if p.Enabled && p.Name == seg && p.Value != "" {
				segs[i] = url.PathEscape(p.Value)
				break
			}
		}
	}
	return raw[:start] + strings.Join(segs, "/") + raw[end:]
}
