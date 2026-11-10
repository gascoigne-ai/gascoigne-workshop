package clientconfig

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

const yamlConfig = `servers:
  - name: Local REST
    protocol: rest
    url: http://localhost:8081
    auth:
      type: bearer
      token: secret
collections:
  - name: Demo
    vars:
      baseUrl: http://localhost:8081
    requests:
      - name: Echo
        protocol: rest
        method: POST
        url: "{{baseUrl}}/echo"
        headers:
          X-Trace: "1"
        body_type: json
        body: '{"hello":"world"}'
`

const jsonConfig = `{
  "servers": [{"name": "Local REST", "protocol": "rest", "url": "http://localhost:8081",
    "auth": {"type": "bearer", "token": "secret"}}],
  "collections": [{"name": "Demo", "vars": {"baseUrl": "http://localhost:8081"},
    "requests": [{"name": "Echo", "protocol": "rest", "method": "POST", "url": "{{baseUrl}}/echo",
      "headers": {"X-Trace": "1"}, "body_type": "json", "body": "{\"hello\":\"world\"}"}]}]
}`

const tomlConfig = `[[servers]]
name = "Local REST"
protocol = "rest"
url = "http://localhost:8081"
[servers.auth]
type = "bearer"
token = "secret"

[[collections]]
name = "Demo"
[collections.vars]
baseUrl = "http://localhost:8081"
[[collections.requests]]
name = "Echo"
protocol = "rest"
method = "POST"
url = "{{baseUrl}}/echo"
body_type = "json"
body = '{"hello":"world"}'
[collections.requests.headers]
X-Trace = "1"
`

func wantConfig() Config {
	return Config{
		Servers: []Server{{
			Name: "Local REST", Protocol: "rest", URL: "http://localhost:8081",
			Auth: model.Auth{Type: model.AuthBearer, Token: "secret"},
		}},
		Collections: []*model.Collection{{
			Folder: model.Folder{Name: "Demo", Requests: []*model.Request{{
				Name: "Echo", Protocol: "rest", Method: "POST", URL: "{{baseUrl}}/echo",
				Headers:  []model.KV{{Name: "X-Trace", Value: "1", Enabled: true}},
				BodyType: model.BodyJSON, Body: `{"hello":"world"}`,
			}}},
			Vars: []model.KV{{Name: "baseUrl", Value: "http://localhost:8081", Enabled: true}},
		}},
	}
}

func TestParseReadsTheSameConfigFromEveryFormat(t *testing.T) {
	tests := []struct {
		file string
		data string
	}{
		{".workshop.yaml", yamlConfig},
		{".workshop.yml", yamlConfig},
		{".workshop.json", jsonConfig},
		{".workshop.toml", tomlConfig},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			got, err := Parse(test.file, []byte(test.data))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if want := wantConfig(); !reflect.DeepEqual(got, want) {
				t.Fatalf("config differs\n got: %+v\nwant: %+v", got, want)
			}
		})
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	tests := []struct {
		name, file, data, wantErr string
	}{
		{"unknown extension", ".workshop.ini", "a=1", "unsupported"},
		{"broken json", ".workshop.json", "{", ".workshop.json"},
		{"broken toml", ".workshop.toml", "= 1", ".workshop.toml"},
		{"unknown key", ".workshop.yaml", "serverz:\n  - name: x\n", "serverz"},
		{"server without url", ".workshop.yaml", "servers:\n  - name: x\n    protocol: rest\n", "url"},
		{"collection without name", ".workshop.yaml", "collections:\n  - requests: []\n", "name"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(test.file, []byte(test.data))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Parse error = %v, want it to mention %q", err, test.wantErr)
			}
		})
	}
}

func TestCollectionBuildsOneTreeFromCollectionsAndServers(t *testing.T) {
	config := wantConfig()
	config.Collections[0].Envs = []model.Environment{{Name: "Dev", Vars: []model.KV{{Name: "token", Value: "x", Enabled: true}}}}
	root := config.Collection()

	if len(root.Folders) != 2 || root.Folders[0].Name != "Demo" || root.Folders[1].Name != ServersName {
		t.Fatalf("folders = %+v, want Demo then %q", root.Folders, ServersName)
	}
	demo := root.Folders[0]
	if len(demo.Vars) != 1 || demo.Vars[0].Name != "baseUrl" || len(demo.Requests) != 1 {
		t.Fatalf("Demo = %+v, want its variables on the folder and its request kept", demo)
	}
	request := root.Folders[1].Requests[0]
	if request.Name != "Local REST" || request.URL != "http://localhost:8081" || request.Auth.Token != "secret" {
		t.Fatalf("server request = %+v", request)
	}
	if len(root.Envs) != 1 || root.Envs[0].Name != "Dev" {
		t.Fatalf("envs = %+v, want the collection's environments on the root", root.Envs)
	}
}

func TestCollectionOfNothingIsEmpty(t *testing.T) {
	if !(Config{}).Empty() || wantConfig().Empty() {
		t.Fatal("Empty must be true only for a config without servers and collections")
	}
}

func TestMergeLetsTheOverrideReplaceEntriesOfTheSameName(t *testing.T) {
	base := Config{
		Servers: []Server{{Name: "A", URL: "http://base-a"}, {Name: "B", URL: "http://base-b"}},
		Collections: []*model.Collection{
			{Folder: model.Folder{Name: "One"}}, {Folder: model.Folder{Name: "Two"}},
		},
	}
	override := Config{
		Servers:     []Server{{Name: "B", URL: "http://over-b"}, {Name: "C", URL: "http://over-c"}},
		Collections: []*model.Collection{{Folder: model.Folder{Name: "Two", Requests: []*model.Request{{Name: "r"}}}}},
	}
	got := Merge(base, override)
	wantServers := []Server{{Name: "A", URL: "http://base-a"}, {Name: "B", URL: "http://over-b"}, {Name: "C", URL: "http://over-c"}}
	if !reflect.DeepEqual(got.Servers, wantServers) {
		t.Fatalf("servers = %+v, want %+v", got.Servers, wantServers)
	}
	if len(got.Collections) != 2 || got.Collections[0].Name != "One" || len(got.Collections[1].Requests) != 1 {
		t.Fatalf("collections = %+v, want One kept and Two replaced in place", got.Collections)
	}
}
