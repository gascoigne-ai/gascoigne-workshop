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
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/miniyaml"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// ServersName is the name of the collection that holds one request per
// configured server.
const ServersName = "Servers"

// Config is what one config file, or a merge of several, defines.
type Config struct {
	Servers     []Server
	Collections []*model.Collection
}

// Server is a named connection: the request that reaches it needs only a
// protocol, a URL and credentials.
type Server struct {
	Name     string
	Protocol string
	URL      string
	Auth     model.Auth
}

// format is how one file extension is read into a generic tree.
type format struct {
	ext    string
	decode func(data []byte) (any, error)
}

var formats = []format{
	{".yaml", miniyaml.Parse},
	{".yml", miniyaml.Parse},
	{".json", decodeJSON},
	{".toml", decodeTOML},
}

func decodeJSON(data []byte) (any, error) {
	var tree any
	return tree, json.Unmarshal(data, &tree)
}

func decodeTOML(data []byte) (any, error) {
	var tree map[string]any
	return tree, toml.Unmarshal(data, &tree)
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

func parseWith(f format, file string, data []byte) (Config, error) {
	tree, err := f.decode(data)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", file, err)
	}
	// One typed schema for every format: re-encode the tree as JSON and
	// decode it strictly, so a misspelled key is an error, not ignored.
	normalized, err := json.Marshal(tree)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", file, err)
	}
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

// AllCollections returns the configured collections followed by one
// collection with a request per server, when there are servers.
func (config Config) AllCollections() []*model.Collection {
	all := append([]*model.Collection(nil), config.Collections...)
	if len(config.Servers) == 0 {
		return all
	}
	servers := &model.Collection{Folder: model.Folder{Name: ServersName}}
	for _, server := range config.Servers {
		servers.Requests = append(servers.Requests, &model.Request{
			Name: server.Name, Protocol: server.Protocol, URL: server.URL, Auth: server.Auth,
		})
	}
	return append(all, servers)
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
		config.Servers = append(config.Servers, Server{
			Name: server.Name, Protocol: server.Protocol, URL: server.URL, Auth: server.Auth.model(),
		})
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

func (auth *fileAuth) model() model.Auth {
	if auth == nil {
		return model.Auth{}
	}
	return model.Auth(*auth)
}

func (collection fileCollection) model() (*model.Collection, error) {
	folder, err := collection.fileFolder.model()
	if err != nil {
		return nil, err
	}
	built := &model.Collection{Vars: folder.Vars}
	folder.Vars = nil // a collection keeps its variables on itself, not on its root folder
	built.Folder = *folder
	for _, env := range collection.Envs {
		built.Envs = append(built.Envs, model.Environment{Name: env.Name, Vars: rows(env.Vars)})
	}
	return built, nil
}

func (folder fileFolder) model() (*model.Folder, error) {
	if folder.Name == "" {
		return nil, fmt.Errorf("a collection or folder needs a name")
	}
	built := &model.Folder{Name: folder.Name, Vars: rows(folder.Vars)}
	for _, child := range folder.Folders {
		childBuilt, err := child.model()
		if err != nil {
			return nil, fmt.Errorf("folder %q: %w", folder.Name, err)
		}
		built.Folders = append(built.Folders, childBuilt)
	}
	for _, request := range folder.Requests {
		if request.Name == "" {
			return nil, fmt.Errorf("folder %q: a request needs a name", folder.Name)
		}
		built.Requests = append(built.Requests, request.model())
	}
	return built, nil
}

func (request fileRequest) model() *model.Request {
	return &model.Request{
		Name: request.Name, Protocol: request.Protocol, Method: request.Method, URL: request.URL,
		Params: rows(request.Params), Headers: rows(request.Headers),
		Body: request.Body, BodyType: request.BodyType, TimeoutSec: request.Timeout,
		Auth: request.Auth.model(),
	}
}

// rows turns a name-to-value map into enabled rows sorted by name, so the
// result does not depend on map order.
func rows(values map[string]string) []model.KV {
	if len(values) == 0 {
		return nil
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	kvs := make([]model.KV, len(names))
	for i, name := range names {
		kvs[i] = model.KV{Name: name, Value: values[name], Enabled: true}
	}
	return kvs
}

// Merge returns base with every server and collection of override added;
// one with the same name as an entry of base replaces it in place.
func Merge(base, override Config) Config {
	return Config{
		Servers:     mergeByName(base.Servers, override.Servers, func(server Server) string { return server.Name }),
		Collections: mergeByName(base.Collections, override.Collections, func(collection *model.Collection) string { return collection.Name }),
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
