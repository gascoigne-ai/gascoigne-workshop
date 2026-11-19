package collection

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// GitHubSample is a few REST calls against a public API, plus one call per
// other protocol so the registry is visible in the UI. The debug build's
// Sample starts from it and the App uses it when no Sample is given, so
// tests may rely on its folder order.
func GitHubSample() model.Collection {
	get := func(name, url string, params ...model.KV) *model.Request {
		return &model.Request{Name: name, Protocol: model.ProtocolREST, Method: "GET", URL: url, Params: params}
	}
	kv := func(name, value, desc string) model.KV {
		return model.KV{Name: name, Value: value, Description: desc, Enabled: true}
	}
	return model.Collection{
		Folder: model.Folder{
			Name: "GitHub REST API",
			Folders: []*model.Folder{
				&model.Folder{Name: "User", Requests: []*model.Request{
					get("Get User", "{{baseUrl}}/users/octocat"),
				}},
				&model.Folder{Name: "Repository", Requests: []*model.Request{
					get("Search Repos", "{{baseUrl}}/search/repositories?q=gioui&order=desc&sort=stars",
						kv("q", "gioui", "repo name"), kv("order", "desc", ""), kv("sort", "stars", "metric")),
					get("Repository Info", "{{baseUrl}}/repos/golang/go"),
					get("Repository Tags", "{{baseUrl}}/repos/golang/go/tags"),
				}},
				&model.Folder{Name: "Realtime", Requests: []*model.Request{
					&model.Request{Name: "Echo", Protocol: model.ProtocolWebSocket, URL: "wss://echo.example.org"},
					&model.Request{Name: "List Tools", Protocol: model.ProtocolMCP, URL: "http://localhost:8080/mcp"},
				}},
			},
		},
		Vars: []model.KV{kv("baseUrl", "https://api.github.com", "")},
	}
}

// SampleStart is the folder and request a sample opens on, its search call:
// the first request of its second folder. ok is false for a collection
// without them, which then opens as any other.
func SampleStart(col *model.Collection) (folder *model.Folder, request *model.Request, ok bool) {
	if len(col.Folders) < 2 || len(col.Folders[1].Requests) == 0 {
		return nil, nil, false
	}
	return col.Folders[1], col.Folders[1].Requests[0], true
}
