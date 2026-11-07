package proto

import (
	"net/url"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Curl renders req as a curl command line that sends the same request.
// Variables must already be expanded. It follows what REST.Send does:
// the same automatic Content-Type, the user's Content-Type winning, and
// credentials from the Auth settings.
func Curl(req model.Request) string {
	args := []string{"curl"}
	if !req.NoRedirects {
		args = append(args, "-L")
	}
	if m := strings.ToUpper(req.Method); m != "" && m != "GET" {
		args = append(args, "-X", m)
	}
	args = append(args, shellQuote(curlURL(req)))
	args = append(args, curlHeaders(req)...)
	args = append(args, curlAuthFlags(req.Auth)...)
	return strings.Join(append(args, curlBody(req)...), " ")
}

// curlURL is the URL with an API key added to its query when needed.
func curlURL(req model.Request) string {
	a := req.Auth
	if a.Type != model.AuthAPIKey || a.KeyIn != model.KeyInQuery || a.KeyName == "" {
		return req.URL
	}
	u, err := url.Parse(req.URL)
	if err != nil {
		return req.URL
	}
	q := u.Query()
	q.Set(a.KeyName, a.KeyValue)
	u.RawQuery = q.Encode()
	return u.String()
}

func curlHeaders(req model.Request) []string {
	var args []string
	userContentType := false
	for _, h := range req.Headers {
		if h.Enabled && h.Name != "" {
			userContentType = userContentType || strings.EqualFold(h.Name, "Content-Type")
			args = append(args, "-H", shellQuote(h.Name+": "+h.Value))
		}
	}
	if _, contentType, _ := encodeBody(req); contentType != "" && !userContentType && !isMultipartOrForm(req) {
		args = append(args, "-H", shellQuote("Content-Type: "+contentType))
	}
	return args
}

// isMultipartOrForm reports bodies curl labels itself (-F, --data-urlencode).
func isMultipartOrForm(req model.Request) bool {
	return req.BodyType == model.BodyForm || req.BodyType == model.BodyMultipart
}

func curlAuthFlags(a model.Auth) []string {
	switch a.Type {
	case model.AuthBearer:
		return []string{"-H", shellQuote("Authorization: Bearer " + a.Token)}
	case model.AuthBasic:
		return []string{"-u", shellQuote(a.User + ":" + a.Password)}
	case model.AuthAPIKey:
		if a.KeyName != "" && a.KeyIn != model.KeyInQuery {
			return []string{"-H", shellQuote(a.KeyName + ": " + a.KeyValue)}
		}
	}
	return nil
}

func curlBody(req model.Request) []string {
	var args []string
	switch req.BodyType {
	case model.BodyJSON, model.BodyText:
		args = append(args, "--data-raw", shellQuote(req.Body))
	case model.BodyForm:
		for _, f := range formFields(req) {
			args = append(args, "--data-urlencode", shellQuote(f.Name+"="+f.Value))
		}
	case model.BodyMultipart:
		for _, f := range formFields(req) {
			args = append(args, "-F", shellQuote(f.Name+"="+f.Value))
		}
	case "":
		if req.Body != "" {
			args = append(args, "--data-raw", shellQuote(req.Body))
		}
	}
	return args
}

// shellQuote wraps s in single quotes for a POSIX shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
