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

// BodyTyped is the text of the body changed: the JSON or text, the path of a
// file body, or the query of a GraphQL body, whichever the body type shows.
type BodyTyped struct{ Text string }

// VarsTyped is the variables of a GraphQL body changed.
type VarsTyped struct{ Text string }

// BodyTypeChosen switches the body to Type. Text and FilePath are what the
// text editor and the file field hold, since each type keeps its own in the
// one Body.
type BodyTypeChosen struct{ Type, Text, FilePath string }

// OperationChosen puts a starter GraphQL query and its variables in the body.
type OperationChosen struct{ Query, Variables string }

func (URLTyped) editMsg()        {}
func (ParamsChanged) editMsg()   {}
func (MethodChosen) editMsg()    {}
func (ProtocolChosen) editMsg()  {}
func (TimeoutSet) editMsg()      {}
func (BodyTyped) editMsg()       {}
func (VarsTyped) editMsg()       {}
func (BodyTypeChosen) editMsg()  {}
func (OperationChosen) editMsg() {}

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
	case BodyTyped:
		req.Body = msg.Text
	case VarsTyped:
		req.GraphQLVars = msg.Text
	case BodyTypeChosen:
		return chooseBodyType(req, msg)
	case OperationChosen:
		req.Body, req.GraphQLVars = msg.Query, msg.Variables
	}
	return req
}

// chooseBodyType keeps one Body for all types: a file body holds the path,
// every other the text, so leaving or entering a file body swaps them.
func chooseBodyType(req model.Request, msg BodyTypeChosen) model.Request {
	if req.BodyType == model.BodyFile && msg.Type != model.BodyFile { // the path is not text to keep
		req.Body = msg.Text
	}
	switch msg.Type {
	case model.BodyFile:
		req.Body = msg.FilePath
	case model.BodyGraphQL:
		req.Body = msg.Text
		if req.Protocol == "rest" && (req.Method == "" || req.Method == "GET") {
			req.Method = "POST" // a query goes in the body
		}
	}
	req.BodyType = msg.Type
	return req
}
