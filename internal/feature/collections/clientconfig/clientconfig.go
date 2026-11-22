// Package clientconfig reads .workshop config files: the collections and
// server connections ("clients") a release build offers instead of the
// built-in sample. One file holds JSON, YAML or TOML, chosen by its
// extension; all three are turned into the same generic tree first, so a
// key means the same in every format.
package clientconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/importers/miniyaml"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// Source reads the client config: the collections and servers a build offers
// when no workspace is open. It lives here, beside Config, so the feature
// root imports no sub-package.
type Source interface {
	Load() (Config, error)
}

// ServersName is the name of the folder that holds one request per
// configured server.
const ServersName = "Servers"

// Config is what one config file, or a merge of several, defines.
type Config struct {
	Servers     []Server
	Collections []*request.Collection
}

// Server is a named connection: the request that reaches it needs only a
// protocol, a URL and credentials.
type Server struct {
	Name     string
	Protocol request.Protocol
	URL      string
	Auth     request.Auth
}

// format is how one file extension is read into JSON, the one shape the
// typed schema is decoded from.
type format struct {
	ext    string
	decode func(data []byte) ([]byte, error)
}

var formats = []format{
	{".yaml", miniyaml.ToJSON},
	{".yml", miniyaml.ToJSON},
	{".json", decodeJSON},
	{".toml", decodeTOML},
}

// decodeJSON reads the document into a tree and writes it out again: the
// schema then sees the same normal form as for the other formats.
func decodeJSON(data []byte) ([]byte, error) {
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return nil, err
	}
	return json.Marshal(tree)
}

func decodeTOML(data []byte) ([]byte, error) {
	var tree map[string]any
	if err := toml.Unmarshal(data, &tree); err != nil {
		return nil, err
	}
	return json.Marshal(tree)
}

// FileNames lists the file names a directory may hold the config under,
// in the order they are looked for.
func FileNames() []string {
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = ".workshop" + f.ext
	}
	return names
}

// Parse reads one config file's content. file only selects the format
// (by extension) and names the file in errors.
func Parse(file string, data []byte) (Config, error) {
	ext := strings.ToLower(filepath.Ext(file))
	for _, f := range formats {
		if f.ext == ext {
			return parseWith(f, file, data)
		}
	}
	return Config{}, fmt.Errorf("parse %s: unsupported config format %q", file, ext)
}

func parseWith(syntax format, file string, data []byte) (Config, error) {
	normalized, err := syntax.decode(data)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", file, err)
	}
	// One typed schema for every format, decoded from the JSON form.
	var schema fileConfig
	decoder := json.NewDecoder(bytes.NewReader(normalized))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&schema); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", file, err)
	}
	config, err := schema.config()
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", file, err)
	}
	return config, nil
}

// RootName is the name of the collection Collection builds.
const RootName = "Clients"

// Empty reports whether the config defines nothing.
func (config Config) Empty() bool {
	return len(config.Servers) == 0 && len(config.Collections) == 0
}

// Collection returns the config as the one collection the window shows:
// every configured collection becomes a folder (its variables on the
// folder), followed by a "Servers" folder with a request per server. The
// environments of all collections are gathered on the root.
func (config Config) Collection() request.Collection {
	root := request.Collection{Folder: request.Folder{Name: RootName}}
	for _, collection := range config.Collections {
		folder := collection.Folder
		folder.Vars = collection.Vars
		root.Folders = append(root.Folders, &folder)
		root.Envs = append(root.Envs, collection.Envs...)
	}
	if len(config.Servers) == 0 {
		return root
	}
	servers := &request.Folder{Name: ServersName}
	for _, server := range config.Servers {
		servers.Requests = append(servers.Requests, &request.Request{
			Name: server.Name, Protocol: server.Protocol, URL: server.URL, Auth: server.Auth,
		})
	}
	root.Folders = append(root.Folders, servers)
	return root
}

type fileConfig struct {
	Servers     []fileServer     `json:"servers"`
	Collections []fileCollection `json:"collections"`
}

type fileServer struct {
	Name     string    `json:"name"`
	Protocol string    `json:"protocol"`
	URL      string    `json:"url"`
	Auth     *fileAuth `json:"auth"`
}

type fileAuth struct {
	Type     string `json:"type"`
	Token    string `json:"token"`
	User     string `json:"user"`
	Password string `json:"password"`
	KeyName  string `json:"key_name"`
	KeyValue string `json:"key_value"`
	KeyIn    string `json:"key_in"`
}

type fileCollection struct {
	fileFolder
	Envs []fileEnv `json:"envs"`
}

type fileEnv struct {
	Name string            `json:"name"`
	Vars map[string]string `json:"vars"`
}

type fileFolder struct {
	Name     string            `json:"name"`
	Vars     map[string]string `json:"vars"`
	Folders  []fileFolder      `json:"folders"`
	Requests []fileRequest     `json:"requests"`
}

type fileRequest struct {
	Name     string            `json:"name"`
	Protocol string            `json:"protocol"`
	Method   string            `json:"method"`
	URL      string            `json:"url"`
	Params   map[string]string `json:"params"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	BodyType string            `json:"body_type"`
	Timeout  int               `json:"timeout"`
	Auth     *fileAuth         `json:"auth"`
}

func (schema fileConfig) config() (Config, error) {
	var config Config
	for _, server := range schema.Servers {
		if server.Name == "" || server.Protocol == "" || server.URL == "" {
			return Config{}, fmt.Errorf("server %q needs a name, a protocol and a url", server.Name)
		}
		protocol, err := request.ParseProtocol(server.Protocol)
		if err != nil {
			return Config{}, fmt.Errorf("server %q: %w", server.Name, err)
		}
		auth, err := server.Auth.model()
		if err != nil {
			return Config{}, fmt.Errorf("server %q: %w", server.Name, err)
		}
		config.Servers = append(config.Servers, Server{Name: server.Name, Protocol: protocol, URL: server.URL, Auth: auth})
	}
	for _, collection := range schema.Collections {
		built, err := collection.model()
		if err != nil {
			return Config{}, err
		}
		config.Collections = append(config.Collections, built)
	}
	return config, nil
}

func (auth *fileAuth) model() (request.Auth, error) {
	if auth == nil {
		return request.Auth{}, nil
	}
	authType, err := request.ParseAuthType(auth.Type)
	if err != nil {
		return request.Auth{}, err
	}
	keyIn, err := request.ParseKeyIn(auth.KeyIn)
	if err != nil {
		return request.Auth{}, err
	}
	return request.Auth{
		Type: authType, Token: auth.Token, User: auth.User, Password: auth.Password,
		KeyName: auth.KeyName, KeyValue: auth.KeyValue, KeyIn: keyIn,
	}, nil
}

func (collection fileCollection) model() (*request.Collection, error) {
	folder, err := collection.fileFolder.model()
	if err != nil {
		return nil, err
	}
	built := &request.Collection{Vars: folder.Vars}
	folder.Vars = nil // a collection keeps its variables on itself, not on its root folder
	built.Folder = *folder
	for _, env := range collection.Envs {
		built.Envs = append(built.Envs, request.Environment{Name: env.Name, Vars: rows(env.Vars)})
	}
	return built, nil
}

func (folder fileFolder) model() (*request.Folder, error) {
	if folder.Name == "" {
		return nil, fmt.Errorf("a collection or folder needs a name")
	}
	built := &request.Folder{Name: folder.Name, Vars: rows(folder.Vars)}
	for _, child := range folder.Folders {
		childBuilt, err := child.model()
		if err != nil {
			return nil, fmt.Errorf("folder %q: %w", folder.Name, err)
		}
		built.Folders = append(built.Folders, childBuilt)
	}
	for _, call := range folder.Requests {
		if call.Name == "" {
			return nil, fmt.Errorf("folder %q: a request needs a name", folder.Name)
		}
		builtRequest, err := call.model()
		if err != nil {
			return nil, fmt.Errorf("folder %q: request %q: %w", folder.Name, call.Name, err)
		}
		built.Requests = append(built.Requests, builtRequest)
	}
	return built, nil
}

func (call fileRequest) model() (*request.Request, error) {
	protocol, err := request.ParseProtocol(call.Protocol)
	if err != nil {
		return nil, err
	}
	bodyType, err := request.ParseBodyType(call.BodyType)
	if err != nil {
		return nil, err
	}
	auth, err := call.Auth.model()
	if err != nil {
		return nil, err
	}
	return &request.Request{
		Name: call.Name, Protocol: protocol, Method: call.Method, URL: call.URL,
		Params: rows(call.Params), Headers: rows(call.Headers),
		Body: call.Body, BodyType: bodyType, TimeoutSec: call.Timeout, Auth: auth,
	}, nil
}

// rows turns a name-to-value map into enabled rows sorted by name, so the
// result does not depend on map order.
func rows(values map[string]string) []request.KV {
	if len(values) == 0 {
		return nil
	}
	names := slices.Sorted(maps.Keys(values))
	kvs := make([]request.KV, len(names))
	for i, name := range names {
		kvs[i] = request.KV{Name: name, Value: values[name], Enabled: true}
	}
	return kvs
}

// Merge returns base with every server and collection of override added;
// one with the same name as an entry of base replaces it in place.
func Merge(base, override Config) Config {
	return Config{
		Servers:     mergeByName(base.Servers, override.Servers, func(server Server) string { return server.Name }),
		Collections: mergeByName(base.Collections, override.Collections, func(collection *request.Collection) string { return collection.Name }),
	}
}

func mergeByName[T any](base, override []T, name func(T) string) []T {
	merged := append([]T(nil), base...)
	for _, entry := range override {
		replaced := false
		for i := range merged {
			if name(merged[i]) == name(entry) {
				merged[i], replaced = entry, true
				break
			}
		}
		if !replaced {
			merged = append(merged, entry)
		}
	}
	return merged
}
