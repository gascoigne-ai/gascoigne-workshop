package proto

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// ParseCurl turns a curl command line into a REST request, the way
// "Copy as cURL" of a browser or an API doc writes it. Flags that do
// not change the request (-s, -k, --compressed...) are skipped. As with
// curl, redirects are followed only with -L.
func ParseCurl(cmd string) (model.Request, error) {
	args, err := shellSplit(cmd)
	if err != nil {
		return model.Request{}, err
	}
	if len(args) == 0 {
		return model.Request{}, errors.New("the command is empty")
	}
	if args[0] != "curl" {
		return model.Request{}, errors.New("not a curl command")
	}
	req := model.Request{Protocol: "rest", NoRedirects: true}
	var data, forms []string
	var user, method string
	head := false
	for i := 1; i < len(args); i++ {
		a := args[i]
		value := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s needs a value", a)
			}
			i++
			return args[i], nil
		}
		if strings.HasPrefix(a, "--") && strings.Contains(a, "=") {
			k, v, _ := strings.Cut(a, "=")
			args = append(args[:i+1], append([]string{v}, args[i+1:]...)...)
			a = k
		}
		var v string
		switch a {
		case "-X", "--request":
			if v, err = value(); err == nil {
				method = strings.ToUpper(v)
			}
		case "-H", "--header":
			if v, err = value(); err == nil {
				name, val, _ := strings.Cut(v, ":")
				req.Headers = append(req.Headers, model.KV{Name: strings.TrimSpace(name), Value: strings.TrimSpace(val), Enabled: true})
			}
		case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii", "--data-urlencode":
			if v, err = value(); err == nil {
				data = append(data, v)
			}
		case "-F", "--form", "--form-string":
			if v, err = value(); err == nil {
				forms = append(forms, v)
			}
		case "-u", "--user":
			user, err = value()
		case "--url":
			req.URL, err = value()
		case "-A", "--user-agent":
			if v, err = value(); err == nil {
				req.Headers = append(req.Headers, model.KV{Name: "User-Agent", Value: v, Enabled: true})
			}
		case "-b", "--cookie":
			if v, err = value(); err == nil {
				req.Headers = append(req.Headers, model.KV{Name: "Cookie", Value: v, Enabled: true})
			}
		case "-e", "--referer":
			if v, err = value(); err == nil {
				req.Headers = append(req.Headers, model.KV{Name: "Referer", Value: v, Enabled: true})
			}
		case "-m", "--max-time":
			if v, err = value(); err == nil {
				if sec, perr := strconv.ParseFloat(v, 64); perr == nil && sec >= 1 {
					req.TimeoutSec = int(sec)
				}
			}
		case "-L", "--location":
			req.NoRedirects = false
		case "-I", "--head":
			head = true
		case "-o", "--output", "-w", "--write-out", "--connect-timeout", "--retry", "-x", "--proxy", "--cacert", "--cert", "--key", "-T", "--upload-file":
			_, err = value() // takes a value, changes nothing here
		default:
			if !strings.HasPrefix(a, "-") && req.URL == "" {
				req.URL = a
			}
		}
		if err != nil {
			return model.Request{}, err
		}
	}
	if req.URL == "" {
		return model.Request{}, errors.New("no URL in the command")
	}
	req.Params = model.ParamsFromURL(req.URL)
	if len(req.Params) == 0 {
		req.Params = nil
	}
	req.Auth = curlAuth(&req, user)
	req.Method, req.BodyType, req.Body, req.Form = curlBodyOf(req.Headers, method, head, data, forms)
	return req, nil
}

// curlAuth takes credentials out of a -u flag or a bearer header.
func curlAuth(req *model.Request, user string) model.Auth {
	if user != "" {
		name, pw, _ := strings.Cut(user, ":")
		return model.Auth{Type: model.AuthBasic, User: name, Password: pw}
	}
	for i, h := range req.Headers {
		if strings.EqualFold(h.Name, "Authorization") && strings.HasPrefix(h.Value, "Bearer ") {
			req.Headers = append(req.Headers[:i], req.Headers[i+1:]...)
			if len(req.Headers) == 0 {
				req.Headers = nil
			}
			return model.Auth{Type: model.AuthBearer, Token: strings.TrimPrefix(h.Value, "Bearer ")}
		}
	}
	return model.Auth{}
}

// curlBodyOf decides the method and the body kind from the data flags.
func curlBodyOf(headers []model.KV, method string, head bool, data, forms []string) (m, bodyType, body string, form []model.KV) {
	contentType := ""
	for _, h := range headers {
		if strings.EqualFold(h.Name, "Content-Type") {
			contentType = strings.ToLower(h.Value)
		}
	}
	switch {
	case len(forms) > 0:
		bodyType = model.BodyMultipart
		for _, f := range forms {
			name, val, _ := strings.Cut(f, "=")
			form = append(form, model.KV{Name: name, Value: val, Enabled: true})
		}
	case len(data) > 0:
		joined := strings.Join(data, "&")
		switch {
		case strings.Contains(contentType, "json") || (contentType == "" && looksJSON(joined)):
			bodyType, body = model.BodyJSON, joined
		case contentType == "" || strings.Contains(contentType, "x-www-form-urlencoded"):
			bodyType = model.BodyForm
			for _, pair := range strings.Split(joined, "&") {
				name, val, _ := strings.Cut(pair, "=")
				form = append(form, model.KV{Name: unescapeForm(name), Value: unescapeForm(val), Enabled: true})
			}
		default:
			bodyType, body = model.BodyText, joined
		}
	default:
		bodyType = model.BodyNone
	}
	switch {
	case method != "":
		m = method
	case head:
		m = "HEAD"
	case bodyType != model.BodyNone:
		m = "POST"
	default:
		m = "GET"
	}
	return m, bodyType, body, form
}

func looksJSON(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func unescapeForm(s string) string {
	if u, err := url.QueryUnescape(s); err == nil {
		return u
	}
	return s
}

// shellSplit splits a command line into words the way a POSIX shell
// does: single quotes, double quotes, $'...' escapes, backslashes and
// line continuations.
func shellSplit(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	r := []rune(s)
	flush := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
		case c == '\\':
			if i+1 < len(r) {
				i++
				if r[i] == '\n' { // line continuation
					continue
				}
				if r[i] == '\r' && i+1 < len(r) && r[i+1] == '\n' {
					i++
					continue
				}
				cur.WriteRune(r[i])
				inWord = true
			}
		case c == '\'':
			inWord = true
			end := indexRune(r, i+1, '\'')
			if end < 0 {
				return nil, errors.New("unterminated quote")
			}
			cur.WriteString(string(r[i+1 : end]))
			i = end
		case c == '$' && i+1 < len(r) && r[i+1] == '\'':
			inWord = true
			j := i + 2
			for ; j < len(r) && r[j] != '\''; j++ {
				if r[j] == '\\' && j+1 < len(r) {
					j++
					switch r[j] {
					case 'n':
						cur.WriteByte('\n')
					case 't':
						cur.WriteByte('\t')
					case 'r':
						cur.WriteByte('\r')
					default:
						cur.WriteRune(r[j])
					}
					continue
				}
				cur.WriteRune(r[j])
			}
			if j >= len(r) {
				return nil, errors.New("unterminated quote")
			}
			i = j
		case c == '"':
			inWord = true
			j := i + 1
			for ; j < len(r) && r[j] != '"'; j++ {
				if r[j] == '\\' && j+1 < len(r) && strings.ContainsRune("\"\\$`", r[j+1]) {
					j++
				}
				cur.WriteRune(r[j])
			}
			if j >= len(r) {
				return nil, errors.New("unterminated quote")
			}
			i = j
		default:
			cur.WriteRune(c)
			inWord = true
		}
	}
	flush()
	return words, nil
}

func indexRune(r []rune, from int, c rune) int {
	for i := from; i < len(r); i++ {
		if r[i] == c {
			return i
		}
	}
	return -1
}
