package grpc

import (
	"errors"
	"reflect"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// filesShown says the pane shows the file lists: the request has files or
// import paths, or the user chose Proto files and has not added one yet.
// Choosing the source alone does not edit the request, so it is kept here.
func (st *state) filesShown(req *request.Request) bool {
	return st.filesChosen || len(req.Proto.Files) > 0 || len(req.Proto.ImportPaths) > 0
}

// problems are what the files were found to have, for the request's
// current source: the answer for an older one says nothing of it.
func (st *state) problems(source request.ProtoSource) []string {
	if !reflect.DeepEqual(st.check.source, source) {
		return nil
	}
	return st.check.problems
}

// Input is the text field of a path row of tab, as the pane last drew it,
// nil before the pane was drawn.
func (c *Component) Input(tab *tabs.Tab, list requestedit.ProtoList, index int) *widget.TextField {
	made, ok := c.states[tab]
	if !ok {
		return nil
	}
	return input(made, tab.Req.Proto, list, index)
}

// input is the text field of a path row, made on first use. The request
// stays the source of truth: a field is seeded from its path, and made
// again when the text no longer stands for the path.
func input(st *state, source request.ProtoSource, list requestedit.ProtoList, index int) *widget.TextField {
	path := protoPaths(source, list)[index]
	key := rowKey{list, index}
	if field, ok := st.inputs[key]; ok && pathFromText(st.dir, field.Text) == path {
		return field
	}
	// A path changed by something other than typing (a removed row before
	// this one, an added file) shows the request's value, not stale text.
	field := new(widget.TextField)
	*field = widget.NewTextField(request.StoredPath(st.dir, path))
	if st.inputs == nil {
		st.inputs = map[rowKey]*widget.TextField{}
	}
	st.inputs[key] = field
	return field
}

// pathFromText is the path a row's text stands for: a relative one is
// read from the collection's folder, an empty one is the empty path.
func pathFromText(dir, text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return request.LoadedPath(dir, text)
}

func protoPaths(source request.ProtoSource, list requestedit.ProtoList) []string {
	if list == requestedit.ImportPaths {
		return source.ImportPaths
	}
	return source.Files
}

// addFile asks for a .proto file and adds it to the request. The answer
// writes into st, taken before the chooser opened: the tab may be closed
// meanwhile.
func (c *Component) addFile(st *state, tab *tabs.Tab) {
	c.deps.Files.Pick("Choose a .proto file", func(path string) {
		st.filesChosen = true
		editRequest(tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: path})
	})
}

// startCheck reads the request's proto files in the background when they
// changed since the last read, so a problem shows while the user edits and
// not only at send. An answer for an older source is not shown, and the
// next build starts a read of the current one. The read asks the request as
// written, with its {{variables}}: it needs no server.
func (c *Component) startCheck(st *state, tab *tabs.Tab) {
	source := tab.Req.Proto
	if len(source.Files) == 0 {
		st.check = check{source: source}
		return
	}
	if st.check.running || reflect.DeepEqual(st.check.source, source) {
		return
	}
	st.check.running = true
	req := tab.Req.Clone()
	dir := st.dir
	c.deps.Methods.List(req, func(_ []string, err error) {
		st.check = check{source: req.Proto, problems: problemsOf(err, dir)}
	})
}

// ProtoPending reports whether tab's request names proto files nobody has
// finished reading, so a capture of the pane waits. It makes no state.
func (c *Component) ProtoPending(tab *tabs.Tab) bool {
	if len(tab.Req.Proto.Files) == 0 {
		return false
	}
	made, ok := c.states[tab]
	if !ok {
		return true
	}
	return made.check.running || !reflect.DeepEqual(made.check.source, tab.Req.Proto)
}

// problemsOf are the lines that tell what is wrong with the files, each a
// located problem with its file relative to the collection, or the error as
// it is when it has no place in a file.
func problemsOf(err error, dir string) []string {
	if err == nil {
		return nil
	}
	parts := []error{err}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		parts = joined.Unwrap()
	}
	problems := make([]string, 0, len(parts))
	for _, part := range parts {
		var located request.ProtoError
		if !errors.As(part, &located) {
			problems = append(problems, part.Error())
			continue
		}
		located.Path = request.StoredPath(dir, located.Path)
		problems = append(problems, located.Error())
	}
	return problems
}
