package proto

import (
	"net/url"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Curl renders req as a curl command line that sends the same request.
// Variables must already be expanded. It follows what REST.Send does:
// the same automatic Content-Type, the user's Content-Type winning, and
// credentials from the Auth settings.
func Curl(req request.Request) string {
	args := []string{"curl"}
	if !req.NoRedirects {
		args = append(args, "-L")
	}
	if m := strings.ToUpper(req.Method); m != "" && m != "GET" {
		args = append(args, "-X", quoteUnlessToken(m))
	}
	args = append(args, shellQuote(curlURL(req)))
	args = append(args, curlHeaders(req)...)
	args = append(args, curlAuthFlags(req.Auth)...)
	return strings.Join(append(args, curlBody(req)...), " ")
}

// curlURL is the URL with an API key added to its query when needed.
func curlURL(req request.Request) string {
	a := req.Auth
	if a.Type != request.AuthAPIKey || a.KeyIn != request.KeyInQuery || a.KeyName == "" {
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

func curlHeaders(req request.Request) []string {
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
func isMultipartOrForm(req request.Request) bool {
	return req.BodyType == request.BodyForm || req.BodyType == request.BodyMultipart
}

func curlAuthFlags(auth request.Auth) []string {
	switch auth.Type {
	case request.AuthBearer:
		return []string{"-H", shellQuote("Authorization: Bearer " + auth.Token)}
	case request.AuthBasic:
		return []string{"-u", shellQuote(auth.User + ":" + auth.Password)}
	case request.AuthAPIKey:
		if auth.KeyName != "" && auth.KeyIn != request.KeyInQuery {
			return []string{"-H", shellQuote(auth.KeyName + ": " + auth.KeyValue)}
		}
	}
	return nil
}

func curlBody(req request.Request) []string {
	var args []string
	switch req.BodyType {
	case request.BodyJSON, request.BodyText:
		args = append(args, "--data-raw", shellQuote(req.Body))
	case request.BodyForm:
		for _, f := range formFields(req) {
			args = append(args, "--data-urlencode", shellQuote(f.Name+"="+f.Value))
		}
	case request.BodyMultipart:
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

// quoteUnlessToken leaves a plain word such as a method name bare and
// quotes anything else for the shell.
func quoteUnlessToken(text string) string {
	for _, r := range text {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return shellQuote(text)
		}
	}
	return text
}

// shellQuote wraps text in single quotes for a POSIX shell.
func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}
