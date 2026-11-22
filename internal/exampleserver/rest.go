package exampleserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	maxDelay    = 10 * time.Second
	minStatus   = 200
	maxStatus   = 599
	indentation = "  "
)

// restHandler routes the REST examples: /echo reflects the request,
// /status/{code} answers with that code, /delay/{ms} answers late and
// /auth wants an Authorization header.
func restHandler() http.Handler {
	mux := http.NewServeMux()
	for _, route := range restRoutes() {
		mux.HandleFunc(route.pattern, route.handler)
	}
	return mux
}

// restRoute is one endpoint of the REST example. The debug sample
// collection has a request for each (sample_test.go keeps them in step).
type restRoute struct {
	pattern string
	handler http.HandlerFunc
}

func restRoutes() []restRoute {
	routes := []restRoute{
		{"/echo", echo},
		{"/status/{code}", status},
		{"/delay/{ms}", delay},
		{"/auth", auth},
	}
	index := func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(writer, "gascoigne example server\n")
		for _, route := range routes {
			fmt.Fprintln(writer, route.pattern)
		}
	}
	return append([]restRoute{{"/{$}", index}}, routes...)
}

func writeJSON(writer http.ResponseWriter, code int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", indentation)
	_ = encoder.Encode(value) // the client went away; there is nobody to tell
}

func echo(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"method":  request.Method,
		"path":    request.URL.Path,
		"query":   request.URL.Query(),
		"headers": request.Header,
		"body":    string(body),
	})
}

func status(writer http.ResponseWriter, request *http.Request) {
	code, err := strconv.Atoi(request.PathValue("code"))
	if err != nil || code < minStatus || code > maxStatus {
		http.Error(writer, fmt.Sprintf("status must be a number from %d to %d", minStatus, maxStatus), http.StatusBadRequest)
		return
	}
	writeJSON(writer, code, map[string]any{"status": code, "text": http.StatusText(code)})
}

func delay(writer http.ResponseWriter, request *http.Request) {
	millis, err := strconv.Atoi(request.PathValue("ms"))
	if err != nil || millis < 0 {
		http.Error(writer, "delay must be a number of milliseconds", http.StatusBadRequest)
		return
	}
	wait := min(time.Duration(millis)*time.Millisecond, maxDelay)
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		writeJSON(writer, http.StatusOK, map[string]any{"delayed_ms": wait.Milliseconds()})
	case <-request.Context().Done():
	}
}

func auth(writer http.ResponseWriter, request *http.Request) {
	header := request.Header.Get("Authorization")
	if header == "" {
		writer.Header().Set("WWW-Authenticate", `Bearer realm="example"`)
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "missing Authorization header"})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"authorization": header})
}
