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

func TestCollectionsAppendsServersAsOneCollection(t *testing.T) {
	config := wantConfig()
	got := config.AllCollections()
	if len(got) != 2 || got[1].Name != ServersName {
		t.Fatalf("collections = %d, want the Demo collection then %q", len(got), ServersName)
	}
	request := got[1].Requests[0]
	if request.Name != "Local REST" || request.URL != "http://localhost:8081" || request.Auth.Token != "secret" {
		t.Fatalf("server request = %+v", request)
	}
}
