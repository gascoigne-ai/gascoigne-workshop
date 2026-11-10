package collection

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// GitHubSample is a few REST calls against a public API, plus one call per
// other protocol so the registry is visible in the UI. The debug build's
// Sample starts from it and the App uses it when no Sample is given, so
// tests may rely on its folder order.
func GitHubSample() model.Collection {
	get := func(name, url string, params ...model.KV) *model.Request {
		return &model.Request{Name: name, Protocol: "rest", Method: "GET", URL: url, Params: params}
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
					&model.Request{Name: "Echo", Protocol: "websocket", URL: "wss://echo.example.org"},
					&model.Request{Name: "List Tools", Protocol: "mcp", URL: "http://localhost:8080/mcp"},
				}},
			},
		},
		Vars: []model.KV{kv("baseUrl", "https://api.github.com", "")},
	}
}
