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

// AuthTypeChosen sets the authentication type.
type AuthTypeChosen struct{ Type string }

// AuthFieldSet sets one credential field, named by its id: token, user,
// password, keyname, keyvalue or keyin.
type AuthFieldSet struct{ Field, Value string }

// TestsTyped is the assertions text changed.
type TestsTyped struct{ Text string }

// NoRedirectsToggled flips whether redirects are left unfollowed.
type NoRedirectsToggled struct{}

// InsecureTLSToggled flips whether certificate errors are ignored.
type InsecureTLSToggled struct{}

// ProtoList names one list of paths of a gRPC request's proto source.
type ProtoList string

// The lists of model.ProtoSource.
const (
	ProtoFiles  ProtoList = "files"
	ImportPaths ProtoList = "import paths"
)

// ProtoPathAdded appends Path to a list; "" is a row still to be typed.
type ProtoPathAdded struct {
	List ProtoList
	Path string
}

// ProtoPathSet replaces the path at Index of a list.
type ProtoPathSet struct {
	List  ProtoList
	Index int
	Path  string
}

// ProtoPathRemoved removes the path at Index of a list.
type ProtoPathRemoved struct {
	List  ProtoList
	Index int
}

// ReflectionChosen drops the proto files and import paths, so the methods
// come from the server's reflection again.
type ReflectionChosen struct{}

func (URLTyped) editMsg()           {}
func (ProtoPathAdded) editMsg()     {}
func (ProtoPathSet) editMsg()       {}
func (ProtoPathRemoved) editMsg()   {}
func (ReflectionChosen) editMsg()   {}
func (ParamsChanged) editMsg()      {}
func (MethodChosen) editMsg()       {}
func (ProtocolChosen) editMsg()     {}
func (TimeoutSet) editMsg()         {}
func (BodyTyped) editMsg()          {}
func (VarsTyped) editMsg()          {}
func (BodyTypeChosen) editMsg()     {}
func (OperationChosen) editMsg()    {}
func (AuthTypeChosen) editMsg()     {}
func (AuthFieldSet) editMsg()       {}
func (TestsTyped) editMsg()         {}
func (NoRedirectsToggled) editMsg() {}
func (InsecureTLSToggled) editMsg() {}

// authFields is where each credential field named in AuthFieldSet lives.
var authFields = map[string]func(*model.Auth) *string{
	"token":    func(auth *model.Auth) *string { return &auth.Token },
	"user":     func(auth *model.Auth) *string { return &auth.User },
	"password": func(auth *model.Auth) *string { return &auth.Password },
	"keyname":  func(auth *model.Auth) *string { return &auth.KeyName },
	"keyvalue": func(auth *model.Auth) *string { return &auth.KeyValue },
	"keyin":    func(auth *model.Auth) *string { return &auth.KeyIn },
}

// authField is the place of the credential field called id. An unknown id
// is a mistake in the caller's table, not user input, so it panics.
func authField(auth *model.Auth, id string) *string {
	place, ok := authFields[id]
	if !ok {
		panic("editor: unknown auth field " + id)
	}
	return place(auth)
}

// AuthValue is the credential field of auth named by id.
func AuthValue(auth model.Auth, field string) string { return *authField(&auth, field) }

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
	case AuthTypeChosen:
		req.Auth.Type = msg.Type
	case AuthFieldSet:
		*authField(&req.Auth, msg.Field) = msg.Value // req is a copy, and Auth holds only strings
	case TestsTyped:
		req.Tests = msg.Text
	case NoRedirectsToggled:
		req.NoRedirects = !req.NoRedirects
	case InsecureTLSToggled:
		req.InsecureTLS = !req.InsecureTLS
	case ProtoPathAdded:
		list := protoList(&req.Proto, msg.List)
		*list = append(append([]string(nil), *list...), msg.Path)
	case ProtoPathSet:
		list := protoList(&req.Proto, msg.List)
		if msg.Index >= 0 && msg.Index < len(*list) {
			*list = append([]string(nil), *list...)
			(*list)[msg.Index] = msg.Path
		}
	case ProtoPathRemoved:
		list := protoList(&req.Proto, msg.List)
		if msg.Index >= 0 && msg.Index < len(*list) {
			*list = withoutIndex(*list, msg.Index)
		}
	case ReflectionChosen:
		req.Proto = model.ProtoSource{}
	}
	return req
}

// protoLists is where each list named in the proto messages lives.
var protoLists = map[ProtoList]func(*model.ProtoSource) *[]string{
	ProtoFiles:  func(source *model.ProtoSource) *[]string { return &source.Files },
	ImportPaths: func(source *model.ProtoSource) *[]string { return &source.ImportPaths },
}

// protoList is the list called name. An unknown name is a mistake in the
// caller, not user input, so it panics.
func protoList(source *model.ProtoSource, name ProtoList) *[]string {
	place, ok := protoLists[name]
	if !ok {
		panic("editor: unknown proto list " + string(name))
	}
	return place(source)
}

// withoutIndex is a new slice of paths without the one at index; an empty
// result is nil, so an emptied list is not saved.
func withoutIndex(paths []string, index int) []string {
	kept := append(append([]string(nil), paths[:index]...), paths[index+1:]...)
	if len(kept) == 0 {
		return nil
	}
	return kept
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
