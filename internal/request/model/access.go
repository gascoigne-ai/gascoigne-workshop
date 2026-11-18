package model

import (
	"path/filepath"
	"strings"
)

// Access is something a request does on this machine that its author
// may not have meant the user to allow: run a command or read a file.
type Access struct {
	Kind    string // AccessCommand or AccessFile
	Subject string // the command line, or the absolute path
}

// The kinds of Access.
const (
	AccessCommand = "command"
	AccessFile    = "file"
)

// Key is how a trust list remembers the access.
func (access Access) Key() string { return access.Kind + ":" + access.Subject }

const stdioPrefix = "stdio:" // the URL of an MCP server that runs as a command

// Accesses lists what sending or connecting req (already expanded) does on
// this machine: the command of a stdio MCP server, and the files a
// multipart field or a file body reads. A file inside folder is the
// collection's own and is left out; folder "" means none is.
func (r Request) Accesses(folder string) []Access {
	var out []Access
	if command, ok := strings.CutPrefix(r.URL, stdioPrefix); ok && r.Protocol.Info().RunsCommand {
		out = append(out, Access{AccessCommand, strings.TrimSpace(command)})
	}
	switch r.BodyType {
	case BodyFile:
		if path := strings.TrimSpace(r.Body); path != "" {
			out = appendFile(out, path, folder)
		}
	case BodyMultipart:
		for _, field := range r.Form {
			if !field.Enabled {
				continue
			}
			if path, ok := strings.CutPrefix(field.Value, "@"); ok && path != "" && !strings.HasPrefix(path, "@") {
				out = appendFile(out, path, folder)
			}
		}
	}
	return out
}

func appendFile(out []Access, path, folder string) []Access {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = filepath.Clean(path)
	}
	if folder != "" {
		if relative, err := filepath.Rel(folder, absolute); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return out
		}
	}
	return append(out, Access{AccessFile, absolute})
}
