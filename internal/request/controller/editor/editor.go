// Package editor holds the rules of editing a request: each kind of edit is
// a message and Update returns the request after it.
package editor

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Msg is one edit of a request.
type Msg interface{ editMsg() }

// URLTyped is the URL field changed to Text; the Params table follows.
type URLTyped struct{ Text string }

// ParamsChanged says the Params table changed; the URL's query follows.
type ParamsChanged struct{}

// MethodChosen sets the method.
type MethodChosen struct{ Method string }

// ProtocolChosen sets the protocol.
type ProtocolChosen struct{ Protocol string }

// TimeoutSet sets the timeout in seconds; 0 is the default.
type TimeoutSet struct{ Seconds int }

func (URLTyped) editMsg()       {}
func (ParamsChanged) editMsg()  {}
func (MethodChosen) editMsg()   {}
func (ProtocolChosen) editMsg() {}
func (TimeoutSet) editMsg()     {}

// Update returns req after msg. The request it is given is not changed: a
// slice or map an edit changes is replaced, not written into.
func Update(req model.Request, msg Msg) model.Request {
	switch msg := msg.(type) {
	case URLTyped:
		req.URL = msg.Text
		req.Params = model.MergeParams(req.Params, req.URL)
	case ParamsChanged:
		req.URL = model.WithParams(req.URL, req.Params)
	case MethodChosen:
		req.Method = msg.Method
	case ProtocolChosen:
		req.Protocol = msg.Protocol
		if req.Protocol == "rest" && (req.Method == "" || strings.Contains(req.Method, "/")) { // "" or a gRPC method
			req.Method = "GET"
		}
	case TimeoutSet:
		req.TimeoutSec = msg.Seconds
	}
	return req
}
