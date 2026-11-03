package ui

import "github.com/gascoigne-ai/gascoigne-workshop/internal/model"

// SampleCollection is the collection shown until saving and loading
// collections exists: a few REST calls against a public API, plus one
// call per other protocol so the registry is visible in the UI.
func SampleCollection() model.Collection {
	get := func(name, url string, params ...model.KV) model.Request {
		return model.Request{Name: name, Protocol: "rest", Method: "GET", URL: url, Params: params}
	}
	kv := func(name, value, desc string) model.KV {
		return model.KV{Name: name, Value: value, Description: desc, Enabled: true}
	}
	return model.Collection{
		Folder: model.Folder{
			Name: "GitHub REST API",
			Folders: []model.Folder{
				{Name: "User", Requests: []model.Request{
					get("Get User", "{{baseUrl}}/users/octocat"),
				}},
				{Name: "Repository", Requests: []model.Request{
					get("Search Repos", "{{baseUrl}}/search/repositories?q=gioui&order=desc&sort=stars",
						kv("q", "gioui", "repo name"), kv("order", "desc", ""), kv("sort", "stars", "metric")),
					get("Repository Info", "{{baseUrl}}/repos/golang/go"),
					get("Repository Tags", "{{baseUrl}}/repos/golang/go/tags"),
				}},
				{Name: "Realtime", Requests: []model.Request{
					{Name: "Echo", Protocol: "websocket", URL: "wss://echo.example.org"},
					{Name: "List Tools", Protocol: "mcp", URL: "http://localhost:8080/mcp"},
				}},
			},
		},
		Vars: []model.KV{kv("baseUrl", "https://api.github.com", "")},
	}
}
