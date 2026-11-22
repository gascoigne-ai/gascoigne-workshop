// Package store keeps a collection on disk as a tree of small JSON
// files: one directory per folder, one file per request.
//
//	folder.json        {"name", "vars" (root only), "order": ["a.req.json", "sub/"]}
//	a.req.json         one request
//	sub/folder.json    a nested folder
//	environments/x.env.json   one environment (root folder only; a root folder
//	                          named "Environments" is saved as environments-2)
//	.secrets.json      credentials and secret variable values, by file path; mode 0600; listed in .gitignore
//
// Files are written with a stable key order and indentation, so a
// version-control diff shows only what changed. Files the store did not
// write are never touched: saving and Delete remove only files it wrote.
// File names are slugs made unique with -2, -3; the real names live
// inside the files.
//
// A tree of small files rather than one collection file keeps the diff of
// an edit to the request it touched. Credentials stay out of the request
// files so the tree can be committed.
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/userdir"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/fileversion"
)

const (
	envDir      = "environments"
	envSuffix   = ".env.json"
	folderFile  = "folder.json"
	reqSuffix   = ".req.json"
	secretsFile = ".secrets.json"
)

type kvFile struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Secret      bool   `json:"secret,omitempty"`  // the value is in .secrets.json
	Literal     bool   `json:"literal,omitempty"` // a captured value, never expanded
}

type envFile struct {
	Version int      `json:"version"`
	Name    string   `json:"name"`
	Vars    []kvFile `json:"vars,omitempty"`
}

type authFile struct {
	Type    string `json:"type,omitempty"`
	User    string `json:"user,omitempty"`
	KeyName string `json:"keyName,omitempty"`
	KeyIn   string `json:"keyIn,omitempty"`
}

type requestFile struct {
	Version     int      `json:"version"`
	Name        string   `json:"name"`
	Protocol    string   `json:"protocol"`
	Method      string   `json:"method,omitempty"`
	URL         string   `json:"url"`
	Params      []kvFile `json:"params,omitempty"`
	Headers     []kvFile `json:"headers,omitempty"`
	BodyType    string   `json:"bodyType,omitempty"`
	Body        string   `json:"body,omitempty"`
	Form        []kvFile `json:"form,omitempty"`
	Auth        authFile `json:"auth,omitempty"`
	TimeoutSec  int      `json:"timeoutSec,omitempty"`
	NoRedirects bool     `json:"noRedirects,omitempty"`
	InsecureTLS bool     `json:"insecureTls,omitempty"`
	GraphQLVars string   `json:"graphqlVars,omitempty"`
	Tests       string   `json:"tests,omitempty"`
	ProtoFiles  []string `json:"protoFiles,omitempty"`  // gRPC: relative to the collection when inside it
	ImportPaths []string `json:"importPaths,omitempty"` // gRPC: where the imports of ProtoFiles are found
}

type folderMeta struct {
	Version int      `json:"version"`
	Name    string   `json:"name"`
	Vars    []kvFile `json:"vars,omitempty"`
	Order   []string `json:"order,omitempty"`
}

// secret holds the credentials of one request.
type secret struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
	KeyValue string `json:"keyValue,omitempty"`

	Vars map[string]string `json:"vars,omitempty"` // secret variables of an environment, by name
}

// secretsV1 is .secrets.json from version 1: the secrets of each file, by
// its path in the collection, behind the version. Version 0 was the bare map.
type secretsV1 struct {
	Version int               `json:"version"`
	Files   map[string]secret `json:"files"`
}

// The readers of each file by version (see the fileversion package doc and
// architecture/persistence.md); versions 0 and 1 of
// folder, request and environment files share a layout.
var (
	folderReaders  = []func([]byte) (folderMeta, error){fileversion.Plain[folderMeta], fileversion.Plain[folderMeta]}
	requestReaders = []func([]byte) (requestFile, error){fileversion.Plain[requestFile], fileversion.Plain[requestFile]}
	envReaders     = []func([]byte) (envFile, error){fileversion.Plain[envFile], fileversion.Plain[envFile]}
	secretsReaders = []func([]byte) (map[string]secret, error){fileversion.Plain[map[string]secret], readSecretsV1}
)

// The versions Save writes.
var (
	folderVersion  = len(folderReaders) - 1
	requestVersion = len(requestReaders) - 1
	envVersion     = len(envReaders) - 1
	secretsVersion = len(secretsReaders) - 1
)

func readSecretsV1(data []byte) (map[string]secret, error) {
	wrapped, err := fileversion.Plain[secretsV1](data)
	return wrapped.Files, err
}

func (s secret) empty() bool {
	return s.Token == "" && s.Password == "" && s.KeyValue == "" && len(s.Vars) == 0
}

func toKV(rows []request.KV) []kvFile {
	var out []kvFile
	for _, r := range rows {
		out = append(out, kvFile(r))
	}
	return out
}

func fromKV(rows []kvFile) []request.KV {
	var out []request.KV
	for _, r := range rows {
		out = append(out, request.KV(r))
	}
	return out
}

// varsToFile is the rows of a collection or folder as saved in its
// folder.json: the value of a secret variable goes to secrets under
// key (the file's path) instead.
func varsToFile(rows []request.KV, key string, secrets map[string]secret) []kvFile {
	out := toKV(rows)
	for i, row := range rows {
		if !row.Secret {
			continue
		}
		sec := secrets[key]
		if sec.Vars == nil {
			sec.Vars = map[string]string{}
		}
		sec.Vars[row.Name] = row.Value
		secrets[key] = sec
		out[i].Value = ""
	}
	return out
}

// varsFromFile reads rows saved by varsToFile, putting back secret values.
func varsFromFile(rows []kvFile, sec secret) []request.KV {
	out := fromKV(rows)
	for i, row := range rows {
		if value, ok := sec.Vars[row.Name]; row.Secret && ok { // an older file still holds its own value
			out[i].Value = value
		}
	}
	return out
}

// slug turns a name into a file name: lower-case letters and digits
// joined by single dashes.
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		} else {
			dash = true
		}
	}
	if b.Len() == 0 {
		return "item"
	}
	return b.String()
}

// unique returns base, or base-2, base-3... when base is taken.
func unique(base string, taken map[string]bool) string {
	name := base
	for i := 2; taken[strings.ToLower(name)]; i++ {
		name = fmt.Sprintf("%s-%d", base, i)
	}
	taken[strings.ToLower(name)] = true
	return name
}

// Save writes c below dir, creating it, and removes the request and
// folder files of items that no longer exist.
func (Files) Save(dir string, collection request.Collection) error {
	secrets := map[string]secret{}
	meta := folderMeta{Version: folderVersion, Name: collection.Name, Vars: varsToFile(collection.Vars, folderFile, secrets)}
	if err := saveFolder(absoluteDir(dir), dir, "", &collection.Folder, meta, secrets); err != nil {
		return err
	}
	if err := saveEnvs(dir, collection.Envs, secrets); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, secretsFile), secretsV1{Version: secretsVersion, Files: secrets}, len(secrets) == 0); err != nil {
		return err
	}
	return ensureIgnore(dir)
}

// saveFolder writes f into dir, which is rel below the collection's root.
func saveFolder(root, dir, rel string, folder *request.Folder, meta folderMeta, secrets map[string]secret) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	taken := map[string]bool{}
	keep := map[string]bool{folderFile: true}
	for _, r := range folder.Requests {
		file := unique(slug(r.Name), taken) + reqSuffix
		keep[file] = true
		meta.Order = append(meta.Order, file)
		rf, sec := requestToFile(r, root)
		if !sec.empty() {
			secrets[path.Join(rel, file)] = sec
		}
		if err := writeJSON(filepath.Join(dir, file), rf, false); err != nil {
			return err
		}
	}
	if rel == "" {
		taken[envDir] = true // a folder must not take the environments directory
	}
	for _, sub := range folder.Folders {
		name := unique(slug(sub.Name), taken)
		keep[name] = true
		meta.Order = append(meta.Order, name+"/")
		if err := saveFolder(root, filepath.Join(dir, name), path.Join(rel, name), sub, folderMeta{Version: folderVersion, Name: sub.Name, Vars: varsToFile(sub.Vars, path.Join(rel, name, folderFile), secrets)}, secrets); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(dir, folderFile), meta, false); err != nil {
		return err
	}
	return prune(dir, keep, reqSuffix)
}

// saveEnvs writes one file per environment under environments/, keeping
// the values of secret variables out of them. The order of the
// environments is the order of their file names, so they are numbered.
func saveEnvs(dir string, envs []request.Environment, secrets map[string]secret) error {
	ed := filepath.Join(dir, envDir)
	if len(envs) == 0 {
		return prune(ed, map[string]bool{}, envSuffix)
	}
	if err := os.MkdirAll(ed, 0o755); err != nil {
		return err
	}
	taken := map[string]bool{}
	keep := map[string]bool{}
	for i, e := range envs {
		file := fmt.Sprintf("%02d-%s%s", i+1, unique(slug(e.Name), taken), envSuffix)
		keep[file] = true
		ef := envFile{Version: envVersion, Name: e.Name, Vars: varsToFile(e.Vars, path.Join(envDir, file), secrets)}
		if err := writeJSON(filepath.Join(ed, file), ef, false); err != nil {
			return err
		}
	}
	return prune(ed, keep, envSuffix)
}

// loadEnvs reads the environments in file name order.
func loadEnvs(dir string, secrets map[string]secret) ([]request.Environment, error) {
	ed := filepath.Join(dir, envDir)
	entries, err := os.ReadDir(ed)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var envs []request.Environment
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), envSuffix) {
			continue
		}
		ef, err := readJSON(filepath.Join(ed, e.Name()), envReaders)
		if err != nil {
			return nil, err
		}
		env := request.Environment{Name: ef.Name, Vars: varsFromFile(ef.Vars, secrets[path.Join(envDir, e.Name())])}
		envs = append(envs, env)
	}
	return envs, nil
}

// prune removes request files and folders (directories with a
// folder.json) that are not in keep.
func prune(dir string, keep map[string]bool, suffix string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if keep[e.Name()] {
			continue
		}
		p := filepath.Join(dir, e.Name())
		switch {
		case !e.IsDir() && strings.HasSuffix(e.Name(), suffix):
			if err := os.Remove(p); err != nil {
				return err
			}
		case e.IsDir() && fileExists(filepath.Join(p, folderFile)):
			if err := os.RemoveAll(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// absoluteDir is the collection's folder as the root that proto paths are
// stored relative to: absolute, since in memory they are.
func absoluteDir(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil { // no working directory: paths inside dir stay absolute
		return dir
	}
	return abs
}

// requestToFile is the file of r in a collection kept in root; proto paths
// inside root are saved relative to it.
func requestToFile(call *request.Request, root string) (requestFile, secret) {
	proto := call.Proto.Stored(root)
	return requestFile{
		Version: requestVersion, Name: call.Name, Protocol: string(call.Protocol), Method: call.Method, URL: call.URL,
		Params: toKV(call.Params), Headers: toKV(call.Headers),
		BodyType: string(call.BodyType), Body: call.Body, Form: toKV(call.Form),
		Auth:       authFile{Type: string(call.Auth.Type), User: call.Auth.User, KeyName: call.Auth.KeyName, KeyIn: string(call.Auth.KeyIn)},
		TimeoutSec: call.TimeoutSec, NoRedirects: call.NoRedirects, InsecureTLS: call.InsecureTLS, GraphQLVars: call.GraphQLVars, Tests: call.Tests,
		ProtoFiles: proto.Files, ImportPaths: proto.ImportPaths,
	}, secret{Token: call.Auth.Token, Password: call.Auth.Password, KeyValue: call.Auth.KeyValue}
}

// fileToRequest is the request of a file in a collection kept in root. A
// protocol, body type, auth type or key location it does not know is an
// error.
func fileToRequest(file requestFile, secrets secret, root string) (*request.Request, error) {
	protocol, err := request.ParseProtocol(file.Protocol)
	if err != nil {
		return nil, err
	}
	bodyType, err := request.ParseBodyType(file.BodyType)
	if err != nil {
		return nil, err
	}
	authType, err := request.ParseAuthType(file.Auth.Type)
	if err != nil {
		return nil, err
	}
	keyIn, err := request.ParseKeyIn(file.Auth.KeyIn)
	if err != nil {
		return nil, err
	}
	return &request.Request{
		Name: file.Name, Protocol: protocol, Method: file.Method, URL: file.URL,
		Params: fromKV(file.Params), Headers: fromKV(file.Headers),
		BodyType: bodyType, Body: file.Body, Form: fromKV(file.Form),
		Auth: request.Auth{Type: authType, User: file.Auth.User, KeyName: file.Auth.KeyName, KeyIn: keyIn,
			Token: secrets.Token, Password: secrets.Password, KeyValue: secrets.KeyValue},
		TimeoutSec: file.TimeoutSec, NoRedirects: file.NoRedirects, InsecureTLS: file.InsecureTLS, GraphQLVars: file.GraphQLVars, Tests: file.Tests,
		Proto: request.ProtoSource{Files: file.ProtoFiles, ImportPaths: file.ImportPaths}.Loaded(root),
	}, nil
}

// writeJSON writes v atomically, and only when the content changed.
// With remove set it deletes the file instead.
func writeJSON(name string, value any, remove bool) error {
	if remove {
		if err := os.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if filepath.Base(name) == secretsFile {
		perm = 0o600 // credentials: the owner only
	}
	if old, err := os.ReadFile(name); err == nil && bytes.Equal(old, buf.Bytes()) {
		if perm == 0o600 { // a file written world-readable by an older version
			return os.Chmod(name, perm)
		}
		return nil
	}
	return userdir.WriteFileAtomic(name, buf.Bytes(), perm)
}

// ensureIgnore makes sure .gitignore lists the secrets file.
func ensureIgnore(dir string) error {
	p := filepath.Join(dir, ".gitignore")
	old, err := os.ReadFile(p)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, l := range strings.Split(string(old), "\n") {
		if strings.TrimSpace(l) == secretsFile {
			return nil
		}
	}
	if len(old) > 0 && !bytes.HasSuffix(old, []byte("\n")) {
		old = append(old, '\n')
	}
	return os.WriteFile(p, append(old, secretsFile+"\n"...), 0o644)
}

// Load reads the collection below dir. A directory with no files is an
// empty collection.
func (Files) Load(dir string) (request.Collection, error) {
	secrets := map[string]secret{}
	if _, err := os.Stat(filepath.Join(dir, secretsFile)); err == nil {
		read, err := readJSON(filepath.Join(dir, secretsFile), secretsReaders)
		if err != nil {
			return request.Collection{}, err
		}
		if read != nil {
			secrets = read
		}
	}
	var c request.Collection
	meta, err := loadFolder(absoluteDir(dir), dir, "", &c.Folder, secrets)
	if err != nil {
		return request.Collection{}, err
	}
	c.Vars = varsFromFile(meta.Vars, secrets[folderFile])
	if c.Envs, err = loadEnvs(dir, secrets); err != nil {
		return request.Collection{}, err
	}
	return c, nil
}

// readJSON reads the file name with the reader of its version.
func readJSON[T any](name string, readers []func([]byte) (T, error)) (T, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		var zero T
		return zero, err
	}
	value, _, err := fileversion.Read(data, readers)
	if err != nil {
		return value, fmt.Errorf("%s: %w", name, err)
	}
	return value, nil
}

// loadFolder reads f from dir, which is rel below the collection's root.
func loadFolder(root, dir, rel string, folder *request.Folder, secrets map[string]secret) (folderMeta, error) {
	meta, err := readJSON(filepath.Join(dir, folderFile), folderReaders)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return meta, err
	}
	folder.Name = meta.Name
	if rel != "" { // the root's variables are the collection's (Load sets them)
		folder.Vars = varsFromFile(meta.Vars, secrets[path.Join(rel, folderFile)])
	}
	if folder.Name == "" {
		folder.Name = filepath.Base(dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return meta, err
	}
	present := map[string]bool{}
	var extra []string // files and folders the order list does not know, in name order
	for _, e := range entries {
		name := e.Name()
		switch {
		case !e.IsDir() && strings.HasSuffix(name, reqSuffix):
			present[name] = true
		case e.IsDir() && !strings.HasPrefix(name, ".") && fileExists(filepath.Join(dir, name, folderFile)):
			present[name+"/"] = true
		}
	}
	order := slices.DeleteFunc(slices.Clone(meta.Order), func(name string) bool { return !present[name] })
	for n := range present {
		if !slices.Contains(order, n) {
			extra = append(extra, n)
		}
	}
	slices.Sort(extra)
	for _, n := range append(order, extra...) {
		if strings.HasSuffix(n, "/") {
			sub := &request.Folder{}
			if _, err := loadFolder(root, filepath.Join(dir, n), path.Join(rel, strings.TrimSuffix(n, "/")), sub, secrets); err != nil {
				return meta, err
			}
			folder.Folders = append(folder.Folders, sub)
			continue
		}
		name := filepath.Join(dir, n)
		rf, err := readJSON(name, requestReaders)
		if err != nil {
			return meta, err
		}
		call, err := fileToRequest(rf, secrets[path.Join(rel, n)], root)
		if err != nil {
			return meta, fmt.Errorf("%s: %w", name, err)
		}
		folder.Requests = append(folder.Requests, call)
	}
	return meta, nil
}

// Files is the collection store of the file system.
type Files struct{}

var _ collection.Store = Files{}
