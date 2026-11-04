// Package store keeps a collection on disk as a tree of small JSON
// files: one directory per folder, one file per request.
//
//	folder.json        {"name", "vars" (root only), "order": ["a.req.json", "sub/"]}
//	a.req.json         one request
//	sub/folder.json    a nested folder
//	environments/x.env.json   one environment (root folder only)
//	.secrets.json      credentials and secret variable values, by file path; listed in .gitignore
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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
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
	Secret      bool   `json:"secret,omitempty"` // the value is in .secrets.json
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
	Tests       string   `json:"tests,omitempty"`
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
func Save(dir string, c model.Collection) error {
	secrets := map[string]secret{}
	meta := folderMeta{Name: c.Name, Vars: toKV(c.Vars)}
	if err := saveFolder(dir, "", &c.Folder, meta, secrets); err != nil {
		return err
	}
	if err := saveEnvs(dir, c.Envs, secrets); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, secretsFile), secrets, len(secrets) == 0); err != nil {
		return err
	}
	return ensureIgnore(dir)
}

func saveFolder(dir, rel string, f *model.Folder, meta folderMeta, secrets map[string]secret) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	taken := map[string]bool{}
	keep := map[string]bool{folderFile: true}
	for _, r := range f.Requests {
		file := unique(slug(r.Name), taken) + reqSuffix
		keep[file] = true
		meta.Order = append(meta.Order, file)
		rf, sec := requestToFile(r)
		if !sec.empty() {
			secrets[path.Join(rel, file)] = sec
		}
		if err := writeJSON(filepath.Join(dir, file), rf, false); err != nil {
			return err
		}
	}
	for _, sub := range f.Folders {
		name := unique(slug(sub.Name), taken)
		keep[name] = true
		meta.Order = append(meta.Order, name+"/")
		if err := saveFolder(filepath.Join(dir, name), path.Join(rel, name), sub, folderMeta{Name: sub.Name, Vars: toKV(sub.Vars)}, secrets); err != nil {
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

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func requestToFile(r *model.Request) (requestFile, secret) {
	return requestFile{
		Name: r.Name, Protocol: r.Protocol, Method: r.Method, URL: r.URL,
		Params: toKV(r.Params), Headers: toKV(r.Headers),
		BodyType: r.BodyType, Body: r.Body, Form: toKV(r.Form),
		Auth:       authFile{Type: r.Auth.Type, User: r.Auth.User, KeyName: r.Auth.KeyName, KeyIn: r.Auth.KeyIn},
		TimeoutSec: r.TimeoutSec, NoRedirects: r.NoRedirects, InsecureTLS: r.InsecureTLS, Tests: r.Tests,
	}, secret{Token: r.Auth.Token, Password: r.Auth.Password, KeyValue: r.Auth.KeyValue}
}

func fileToRequest(f requestFile, s secret) *model.Request {
	return &model.Request{
		Name: f.Name, Protocol: f.Protocol, Method: f.Method, URL: f.URL,
		Params: fromKV(f.Params), Headers: fromKV(f.Headers),
		BodyType: f.BodyType, Body: f.Body, Form: fromKV(f.Form),
		Auth: model.Auth{Type: f.Auth.Type, User: f.Auth.User, KeyName: f.Auth.KeyName, KeyIn: f.Auth.KeyIn,
			Token: s.Token, Password: s.Password, KeyValue: s.KeyValue},
		TimeoutSec: f.TimeoutSec, NoRedirects: f.NoRedirects, InsecureTLS: f.InsecureTLS, Tests: f.Tests,
	}
}

// writeJSON writes v atomically, and only when the content changed.
// With remove set it deletes the file instead.
func writeJSON(p string, v any, remove bool) error {
	if remove {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	if old, err := os.ReadFile(p); err == nil && bytes.Equal(old, buf.Bytes()) {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(buf.Bytes())
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
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
	meta, err := loadFolder(dir, "", &c.Folder, secrets)
	if err != nil {
		return model.Collection{}, err
	}
	c.Vars = fromKV(meta.Vars)
	if c.Envs, err = loadEnvs(dir, secrets); err != nil {
		return model.Collection{}, err
	}
	return c, nil
}

func readJSON(p string, v any) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	return nil
}

func loadFolder(dir, rel string, f *model.Folder, secrets map[string]secret) (folderMeta, error) {
	var meta folderMeta
	if err := readJSON(filepath.Join(dir, folderFile), &meta); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return meta, err
	}
	f.Name = meta.Name
	if rel != "" { // the root's variables are the collection's (Load sets them)
		f.Vars = fromKV(meta.Vars)
	}
	if f.Name == "" {
		f.Name = filepath.Base(dir)
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
	order := slices.DeleteFunc(slices.Clone(meta.Order), func(n string) bool { return !present[n] })
	for n := range present {
		if !slices.Contains(order, n) {
			extra = append(extra, n)
		}
	}
	slices.Sort(extra)
	for _, n := range append(order, extra...) {
		if strings.HasSuffix(n, "/") {
			sub := &model.Folder{}
			if _, err := loadFolder(filepath.Join(dir, n), path.Join(rel, strings.TrimSuffix(n, "/")), sub, secrets); err != nil {
				return meta, err
			}
			f.Folders = append(f.Folders, sub)
			continue
		}
		var rf requestFile
		if err := readJSON(filepath.Join(dir, n), &rf); err != nil {
			return meta, err
		}
		f.Requests = append(f.Requests, fileToRequest(rf, secrets[path.Join(rel, n)]))
	}
	return meta, nil
}
