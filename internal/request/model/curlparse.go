package model

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseCurl turns a curl command line into a REST request, the way
// "Copy as cURL" of a browser or an API doc writes it. Flags that do
// not change the request (-s, -k, --compressed...) are skipped. As with
// curl, redirects are followed only with -L.
func ParseCurl(cmd string) (Request, error) {
	args, err := ShellSplit(cmd)
	if err != nil {
		return Request{}, err
	}
	if len(args) == 0 {
		return Request{}, errors.New("the command is empty")
	}
	if args[0] != "curl" {
		return Request{}, errors.New("not a curl command")
	}
	args = splitCurlFlags(args)
	parse := curlParse{req: Request{Protocol: ProtocolREST, NoRedirects: true}}
	for i := 1; i < len(args); i++ {
		flag, known := curlFlags[args[i]]
		if !known {
			if !strings.HasPrefix(args[i], "-") && parse.req.URL == "" {
				parse.req.URL = args[i]
			}
			continue
		}
		value := ""
		if flag.takesValue {
			if i+1 >= len(args) {
				return Request{}, fmt.Errorf("%s needs a value", args[i])
			}
			i++
			value = args[i]
		}
		if flag.apply != nil {
			flag.apply(&parse, value)
		}
	}
	req := parse.req
	data, forms, user, method, head := parse.data, parse.forms, parse.user, parse.method, parse.head
	if req.URL == "" {
		return Request{}, errors.New("no URL in the command")
	}
	req.Params = ParamsFromURL(req.URL)
	if len(req.Params) == 0 {
		req.Params = nil
	}
	req.Auth = curlAuth(&req, user)
	req.Method, req.BodyType, req.Body, req.Form = curlBodyOf(req.Headers, method, head, data, forms)
	return req, nil
}

// curlAuth takes credentials out of a -u flag or a bearer header.
func curlAuth(req *Request, user string) Auth {
	if user != "" {
		name, pw, _ := strings.Cut(user, ":")
		return Auth{Type: AuthBasic, User: name, Password: pw}
	}
	for i, h := range req.Headers {
		if strings.EqualFold(h.Name, "Authorization") && strings.HasPrefix(h.Value, "Bearer ") {
			req.Headers = append(req.Headers[:i], req.Headers[i+1:]...)
			if len(req.Headers) == 0 {
				req.Headers = nil
			}
			return Auth{Type: AuthBearer, Token: strings.TrimPrefix(h.Value, "Bearer ")}
		}
	}
	return Auth{}
}

// curlBodyOf decides the method and the body kind from the data flags.
func curlBodyOf(headers []KV, method string, head bool, data, forms []string) (m string, bodyType BodyType, body string, form []KV) {
	contentType := ""
	for _, h := range headers {
		if strings.EqualFold(h.Name, "Content-Type") {
			contentType = strings.ToLower(h.Value)
		}
	}
	switch {
	case len(forms) > 0:
		bodyType = BodyMultipart
		for _, f := range forms {
			name, val, _ := strings.Cut(f, "=")
			form = append(form, KV{Name: name, Value: val, Enabled: true})
		}
	case len(data) > 0:
		joined := strings.Join(data, "&")
		switch {
		case strings.Contains(contentType, "json") || (contentType == "" && startsLikeJSON(joined)):
			bodyType, body = BodyJSON, joined
		case contentType == "" || strings.Contains(contentType, "x-www-form-urlencoded"):
			bodyType = BodyForm
			for _, pair := range strings.Split(joined, "&") {
				name, val, _ := strings.Cut(pair, "=")
				form = append(form, KV{Name: unescapeForm(name), Value: unescapeForm(val), Enabled: true})
			}
		default:
			bodyType, body = BodyText, joined
		}
	default:
		bodyType = BodyNone
	}
	switch {
	case method != "":
		m = method
	case head:
		m = "HEAD"
	case bodyType != BodyNone:
		m = "POST"
	default:
		m = "GET"
	}
	return m, bodyType, body, form
}

// startsLikeJSON is true for text that opens with a brace or a bracket,
// valid or not: data sent by curl is passed on as the user wrote it.
func startsLikeJSON(text string) bool {
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[")
}

func unescapeForm(text string) string {
	if u, err := url.QueryUnescape(text); err == nil {
		return u
	}
	return text
}

// ShellSplit splits a command line into words the way a POSIX shell
// does: single quotes, double quotes, $'...' escapes, backslashes and
// line continuations.
func ShellSplit(text string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	r := []rune(text)
	flush := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	for i := 0; i < len(r); i++ {
		c := r[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			flush()
			continue
		}
		var err error
		switch {
		case c == '\\':
			i, inWord = readBackslash(r, i, &cur, inWord)
		case c == '\'':
			inWord = true
			i, err = readSingleQuoted(r, i, &cur)
		case c == '$' && i+1 < len(r) && r[i+1] == '\'':
			inWord = true
			i, err = readANSIQuoted(r, i, &cur)
		case c == '"':
			inWord = true
			i, err = readDoubleQuoted(r, i, &cur)
		default:
			cur.WriteRune(c)
			inWord = true
		}
		if err != nil {
			return nil, err
		}
	}
	flush()
	return words, nil
}

var errUnterminatedQuote = errors.New("unterminated quote")

// readBackslash handles the escape at r[i] and returns the index of the last
// rune it used. A backslash before a newline is a line continuation and a
// trailing one is dropped; neither starts a word.
func readBackslash(r []rune, i int, cur *strings.Builder, inWord bool) (int, bool) {
	if i+1 >= len(r) {
		return i, inWord
	}
	i++
	if r[i] == '\n' {
		return i, inWord
	}
	if r[i] == '\r' && i+1 < len(r) && r[i+1] == '\n' {
		return i + 1, inWord
	}
	cur.WriteRune(r[i])
	return i, true
}

// readSingleQuoted copies what stands between the quote at r[i] and the next
// one and returns the index of the closing quote.
func readSingleQuoted(r []rune, i int, cur *strings.Builder) (int, error) {
	end := indexRune(r, i+1, '\'')
	if end < 0 {
		return i, errUnterminatedQuote
	}
	cur.WriteString(string(r[i+1 : end]))
	return end, nil
}

// readANSIQuoted reads $'...' starting at the dollar at r[i], where \n, \t
// and \r are escapes and any other escaped rune stands for itself.
func readANSIQuoted(r []rune, i int, cur *strings.Builder) (int, error) {
	j := i + 2
	for ; j < len(r) && r[j] != '\''; j++ {
		if r[j] != '\\' || j+1 >= len(r) {
			cur.WriteRune(r[j])
			continue
		}
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
	}
	if j >= len(r) {
		return i, errUnterminatedQuote
	}
	return j, nil
}

// readDoubleQuoted reads "..." starting at r[i], where a backslash only
// escapes a quote, a backslash, a dollar or a backtick.
func readDoubleQuoted(r []rune, i int, cur *strings.Builder) (int, error) {
	j := i + 1
	for ; j < len(r) && r[j] != '"'; j++ {
		if r[j] == '\\' && j+1 < len(r) && strings.ContainsRune("\"\\$`", r[j+1]) {
			j++
		}
		cur.WriteRune(r[j])
	}
	if j >= len(r) {
		return i, errUnterminatedQuote
	}
	return j, nil
}

func indexRune(runes []rune, from int, char rune) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == char {
			return i
		}
	}
	return -1
}

// curlParse is what the flags of one command line add up to.
type curlParse struct {
	req          Request
	data, forms  []string
	user, method string
	head         bool
}

// curlFlag is one flag: whether the next argument is its value, and what it
// does to the parse. A nil apply is a flag that is read and changes nothing.
type curlFlag struct {
	takesValue bool
	apply      func(parse *curlParse, value string)
}

// curlFlagRow is a flag with all its spellings.
type curlFlagRow struct {
	names []string
	curlFlag
}

// curlFlags maps every spelling of a flag, short and long, to its meaning.
var curlFlags = curlFlagTable([]curlFlagRow{
	{[]string{"-X", "--request"}, curlFlag{true, func(parse *curlParse, value string) { parse.method = strings.ToUpper(value) }}},
	{[]string{"-H", "--header"}, curlFlag{true, func(parse *curlParse, value string) {
		name, val, _ := strings.Cut(value, ":")
		parse.addHeader(strings.TrimSpace(name), strings.TrimSpace(val))
	}}},
	{[]string{"-d", "--data", "--data-raw", "--data-binary", "--data-ascii", "--data-urlencode"},
		curlFlag{true, func(parse *curlParse, value string) { parse.data = append(parse.data, value) }}},
	{[]string{"-F", "--form", "--form-string"},
		curlFlag{true, func(parse *curlParse, value string) { parse.forms = append(parse.forms, value) }}},
	{[]string{"-u", "--user"}, curlFlag{true, func(parse *curlParse, value string) { parse.user = value }}},
	{[]string{"--url"}, curlFlag{true, func(parse *curlParse, value string) { parse.req.URL = value }}},
	{[]string{"-A", "--user-agent"}, curlFlag{true, func(parse *curlParse, value string) { parse.addHeader("User-Agent", value) }}},
	{[]string{"-b", "--cookie"}, curlFlag{true, func(parse *curlParse, value string) { parse.addHeader("Cookie", value) }}},
	{[]string{"-e", "--referer"}, curlFlag{true, func(parse *curlParse, value string) { parse.addHeader("Referer", value) }}},
	{[]string{"-m", "--max-time"}, curlFlag{true, func(parse *curlParse, value string) {
		if sec, err := strconv.ParseFloat(value, 64); err == nil && sec >= 1 {
			parse.req.TimeoutSec = int(sec)
		}
	}}},
	{[]string{"-L", "--location"}, curlFlag{false, func(parse *curlParse, _ string) { parse.req.NoRedirects = false }}},
	{[]string{"-I", "--head"}, curlFlag{false, func(parse *curlParse, _ string) { parse.head = true }}},
	// These take a value and change nothing here.
	{[]string{"-o", "--output", "-w", "--write-out", "--connect-timeout", "--retry", "-x", "--proxy", "--cacert", "--cert", "--key", "-T", "--upload-file"},
		curlFlag{takesValue: true}},
})

func curlFlagTable(rows []curlFlagRow) map[string]curlFlag {
	table := map[string]curlFlag{}
	for _, row := range rows {
		for _, name := range row.names {
			table[name] = row.curlFlag
		}
	}
	return table
}

func (parse *curlParse) addHeader(name, value string) {
	parse.req.Headers = append(parse.req.Headers, KV{Name: name, Value: value, Enabled: true})
}

// splitCurlFlags writes every flag as the parser reads it: "--flag=value"
// as two arguments, and "-XPOST", "-sSL", "-uuser:pw" as single-letter
// flags each followed by its value when it takes one. A "--flag=value"
// of a flag that takes no value drops the value, so it cannot become the
// URL.
func splitCurlFlags(args []string) []string {
	out := []string{args[0]}
	takesValue := false // the previous flag is still waiting for its value
	for _, arg := range args[1:] {
		switch {
		case takesValue: // "-d -x" sends "-x": a value is never a flag
			out = append(out, arg)
			takesValue = false
			continue
		case strings.HasPrefix(arg, "--"):
			name, value, hasValue := strings.Cut(arg, "=")
			out = append(out, name)
			if hasValue && curlFlags[name].takesValue {
				out = append(out, value)
			}
			takesValue = !hasValue && curlFlags[name].takesValue
		case len(arg) > 2 && arg[0] == '-' && arg[1] != '-':
			var parts []string
			parts, takesValue = splitShortCluster(arg[1:])
			out = append(out, parts...)
		default:
			out = append(out, arg)
			takesValue = curlFlags[arg].takesValue
		}
	}
	return out
}

// splitShortCluster splits the letters after a single dash: letters up to
// a value-taking one are flags, and the rest of the word is its value.
// needsValue is true when the last flag takes a value that is the next
// argument.
func splitShortCluster(letters string) (flags []string, needsValue bool) {
	for i, letter := range letters {
		flag := "-" + string(letter)
		flags = append(flags, flag)
		if !curlFlags[flag].takesValue {
			continue
		}
		rest := letters[i+len(string(letter)):]
		if rest == "" {
			return flags, true
		}
		return append(flags, rest), false
	}
	return flags, false
}

// ImportedName names a request imported from a command after its method and
// the path of its URL: "GET /users".
func ImportedName(request Request) string {
	path := request.URL
	if _, rest, ok := strings.Cut(path, "://"); ok {
		path = rest
	}
	path, _, _ = strings.Cut(path, "?")
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[i:]
	} else {
		path = "/"
	}
	return request.Method + " " + path
}
