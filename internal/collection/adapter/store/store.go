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
// write are never touched.
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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
	Name string   `json:"name"`
	Vars []kvFile `json:"vars,omitempty"`
}

type authFile struct {
	Type    string `json:"type,omitempty"`
	User    string `json:"user,omitempty"`
	KeyName string `json:"keyName,omitempty"`
	KeyIn   string `json:"keyIn,omitempty"`
}

type requestFile struct {
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
	Name  string   `json:"name"`
	Vars  []kvFile `json:"vars,omitempty"`
	Order []string `json:"order,omitempty"`
}

// secret holds the credentials of one request.
type secret struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
	KeyValue string `json:"keyValue,omitempty"`

	Vars map[string]string `json:"vars,omitempty"` // secret variables of an environment, by name
}

func (s secret) empty() bool {
	return s.Token == "" && s.Password == "" && s.KeyValue == "" && len(s.Vars) == 0
}

func toKV(rows []model.KV) []kvFile {
	var out []kvFile
	for _, r := range rows {
		out = append(out, kvFile(r))
	}
	return out
}

func fromKV(rows []kvFile) []model.KV {
	var out []model.KV
	for _, r := range rows {
		out = append(out, model.KV(r))
	}
	return out
}

// varsToFile is the rows of a collection or folder as saved in its
// folder.json: the value of a secret variable goes to secrets under
// key (the file's path) instead.
func varsToFile(rows []model.KV, key string, secrets map[string]secret) []kvFile {
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
func varsFromFile(rows []kvFile, sec secret) []model.KV {
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
func Save(dir string, collection model.Collection) error {
	secrets := map[string]secret{}
	meta := folderMeta{Name: collection.Name, Vars: varsToFile(collection.Vars, folderFile, secrets)}
	if err := saveFolder(absoluteDir(dir), dir, "", &collection.Folder, meta, secrets); err != nil {
		return err
	}
	if err := saveEnvs(dir, collection.Envs, secrets); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, secretsFile), secrets, len(secrets) == 0); err != nil {
		return err
	}
	return ensureIgnore(dir)
}

// saveFolder writes f into dir, which is rel below the collection's root.
func saveFolder(root, dir, rel string, folder *model.Folder, meta folderMeta, secrets map[string]secret) error {
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
		if err := saveFolder(root, filepath.Join(dir, name), path.Join(rel, name), sub, folderMeta{Name: sub.Name, Vars: varsToFile(sub.Vars, path.Join(rel, name, folderFile), secrets)}, secrets); err != nil {
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
func saveEnvs(dir string, envs []model.Environment, secrets map[string]secret) error {
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
		ef := envFile{Name: e.Name}
		sec := secret{Vars: map[string]string{}}
		for _, v := range e.Vars {
			kf := kvFile{Name: v.Name, Value: v.Value, Description: v.Description, Enabled: v.Enabled, Secret: v.Secret}
			if v.Secret {
				sec.Vars[v.Name] = v.Value
				kf.Value = ""
			}
			ef.Vars = append(ef.Vars, kf)
		}
		if len(sec.Vars) > 0 {
			secrets[path.Join(envDir, file)] = sec
		}
		if err := writeJSON(filepath.Join(ed, file), ef, false); err != nil {
			return err
		}
	}
	return prune(ed, keep, envSuffix)
}

// loadEnvs reads the environments in file name order.
func loadEnvs(dir string, secrets map[string]secret) ([]model.Environment, error) {
	ed := filepath.Join(dir, envDir)
	entries, err := os.ReadDir(ed)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var envs []model.Environment
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), envSuffix) {
			continue
		}
		var ef envFile
		if err := readJSON(filepath.Join(ed, e.Name()), &ef); err != nil {
			return nil, err
		}
		env := model.Environment{Name: ef.Name}
		sec := secrets[path.Join(envDir, e.Name())]
		for _, kf := range ef.Vars {
			v := model.KV{Name: kf.Name, Value: kf.Value, Description: kf.Description, Enabled: kf.Enabled, Secret: kf.Secret}
			if kf.Secret {
				v.Value = sec.Vars[kf.Name]
			}
			env.Vars = append(env.Vars, v)
		}
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
func requestToFile(request *model.Request, root string) (requestFile, secret) {
	proto := request.Proto.Stored(root)
	return requestFile{
		Name: request.Name, Protocol: string(request.Protocol), Method: request.Method, URL: request.URL,
		Params: toKV(request.Params), Headers: toKV(request.Headers),
		BodyType: string(request.BodyType), Body: request.Body, Form: toKV(request.Form),
		Auth:       authFile{Type: string(request.Auth.Type), User: request.Auth.User, KeyName: request.Auth.KeyName, KeyIn: request.Auth.KeyIn},
		TimeoutSec: request.TimeoutSec, NoRedirects: request.NoRedirects, InsecureTLS: request.InsecureTLS, GraphQLVars: request.GraphQLVars, Tests: request.Tests,
		ProtoFiles: proto.Files, ImportPaths: proto.ImportPaths,
	}, secret{Token: request.Auth.Token, Password: request.Auth.Password, KeyValue: request.Auth.KeyValue}
}

// fileToRequest is the request of a file in a collection kept in root. A
// protocol, body type or auth type it does not know is an error.
func fileToRequest(requestFile requestFile, secret secret, root string) (*model.Request, error) {
	protocol, err := model.ParseProtocol(requestFile.Protocol)
	if err != nil {
		return nil, err
	}
	bodyType, err := model.ParseBodyType(requestFile.BodyType)
	if err != nil {
		return nil, err
	}
	authType, err := model.ParseAuthType(requestFile.Auth.Type)
	if err != nil {
		return nil, err
	}
	return &model.Request{
		Name: requestFile.Name, Protocol: protocol, Method: requestFile.Method, URL: requestFile.URL,
		Params: fromKV(requestFile.Params), Headers: fromKV(requestFile.Headers),
		BodyType: bodyType, Body: requestFile.Body, Form: fromKV(requestFile.Form),
		Auth: model.Auth{Type: authType, User: requestFile.Auth.User, KeyName: requestFile.Auth.KeyName, KeyIn: requestFile.Auth.KeyIn,
			Token: secret.Token, Password: secret.Password, KeyValue: secret.KeyValue},
		TimeoutSec: requestFile.TimeoutSec, NoRedirects: requestFile.NoRedirects, InsecureTLS: requestFile.InsecureTLS, GraphQLVars: requestFile.GraphQLVars, Tests: requestFile.Tests,
		Proto: model.ProtoSource{Files: requestFile.ProtoFiles, ImportPaths: requestFile.ImportPaths}.Loaded(root),
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
	tmp, err := os.CreateTemp(filepath.Dir(name), ".tmp-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(buf.Bytes())
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), name); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
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
func Load(dir string) (model.Collection, error) {
	secrets := map[string]secret{}
	if b, err := os.ReadFile(filepath.Join(dir, secretsFile)); err == nil {
		if err := json.Unmarshal(b, &secrets); err != nil {
			return model.Collection{}, fmt.Errorf("%s: %w", secretsFile, err)
		}
	}
	var c model.Collection
	meta, err := loadFolder(absoluteDir(dir), dir, "", &c.Folder, secrets)
	if err != nil {
		return model.Collection{}, err
	}
	c.Vars = varsFromFile(meta.Vars, secrets[folderFile])
	if c.Envs, err = loadEnvs(dir, secrets); err != nil {
		return model.Collection{}, err
	}
	return c, nil
}

func readJSON(name string, value any) error {
	b, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, value); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// loadFolder reads f from dir, which is rel below the collection's root.
func loadFolder(root, dir, rel string, folder *model.Folder, secrets map[string]secret) (folderMeta, error) {
	var meta folderMeta
	if err := readJSON(filepath.Join(dir, folderFile), &meta); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
			sub := &model.Folder{}
			if _, err := loadFolder(root, filepath.Join(dir, n), path.Join(rel, strings.TrimSuffix(n, "/")), sub, secrets); err != nil {
				return meta, err
			}
			folder.Folders = append(folder.Folders, sub)
			continue
		}
		var rf requestFile
		name := filepath.Join(dir, n)
		if err := readJSON(name, &rf); err != nil {
			return meta, err
		}
		request, err := fileToRequest(rf, secrets[path.Join(rel, n)], root)
		if err != nil {
			return meta, fmt.Errorf("%s: %w", name, err)
		}
		folder.Requests = append(folder.Requests, request)
	}
	return meta, nil
}

// Files is the collection store of the file system.
type Files struct{}

// Load implements port.Store.
func (Files) Load(dir string) (model.Collection, error) { return Load(dir) }

// Save implements port.Store.
func (Files) Save(dir string, collection model.Collection) error { return Save(dir, collection) }

// Create, Rename, Delete and List are the functions of the same names.
func (Files) Create(parent, name string) (string, error) { return Create(parent, name) }
func (Files) Rename(dir, name string) (string, error)    { return Rename(dir, name) }
func (Files) Delete(dir string) error                    { return Delete(dir) }
func (Files) List(parent string) ([]string, error)       { return List(parent) }

var _ port.Store = Files{}
