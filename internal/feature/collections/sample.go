package collection

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"

// GitHubSample is a few REST calls against a public API, plus one call per
// other protocol so the registry is visible in the UI. The debug build's
// Sample starts from it and the App uses it when no Sample is given, so
// tests may rely on its folder order.
func GitHubSample() request.Collection {
	get := func(name, url string, params ...request.KV) *request.Request {
		return &request.Request{Name: name, Protocol: request.ProtocolREST, Method: "GET", URL: url, Params: params}
	}
	kv := func(name, value, desc string) request.KV {
		return request.KV{Name: name, Value: value, Description: desc, Enabled: true}
	}
	return request.Collection{
		Folder: request.Folder{
			Name: "GitHub REST API",
			Folders: []*request.Folder{
				&request.Folder{Name: "User", Requests: []*request.Request{
					get("Get User", "{{baseUrl}}/users/octocat"),
				}},
				&request.Folder{Name: "Repository", Requests: []*request.Request{
					get("Search Repos", "{{baseUrl}}/search/repositories?q=gioui&order=desc&sort=stars",
						kv("q", "gioui", "repo name"), kv("order", "desc", ""), kv("sort", "stars", "metric")),
					get("Repository Info", "{{baseUrl}}/repos/golang/go"),
					get("Repository Tags", "{{baseUrl}}/repos/golang/go/tags"),
				}},
				&request.Folder{Name: "Realtime", Requests: []*request.Request{
					&request.Request{Name: "Echo", Protocol: request.ProtocolWebSocket, URL: "wss://echo.example.org"},
					&request.Request{Name: "List Tools", Protocol: request.ProtocolMCP, URL: "http://localhost:8080/mcp"},
				}},
			},
		},
		Vars: []request.KV{kv("baseUrl", "https://api.github.com", "")},
	}
}

// SampleStart is the folder and request a sample opens on, its search call:
// the first request of its second folder. ok is false for a collection
// without them, which then opens as any other.
func SampleStart(col *request.Collection) (folder *request.Folder, call *request.Request, ok bool) {
	if len(col.Folders) < 2 || len(col.Folders[1].Requests) == 0 {
		return nil, nil, false
	}
	return col.Folders[1], col.Folders[1].Requests[0], true
}
