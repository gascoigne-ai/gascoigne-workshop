package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/clientconfig"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

type fakeClients struct {
	config clientconfig.Config
	err    error
}

func (clients fakeClients) Load() (clientconfig.Config, error) { return clients.config, clients.err }

func demoConfig() clientconfig.Config {
	return clientconfig.Config{Collections: []*model.Collection{{Folder: model.Folder{
		Name: "Demo", Requests: []*model.Request{{Name: "Echo", Protocol: "rest", Method: "GET", URL: "http://x"}},
	}}}}
}

func TestStartWithoutAWorkspaceOpensTheClientConfig(t *testing.T) {
	app, err := NewApp(Options{Sample: collection.GitHubSample, Clients: fakeClients{config: demoConfig()}})
	if err != nil {
		t.Fatal(err)
	}
	if app.col.Name != clientconfig.RootName || selectedName(app) != "Echo" {
		t.Fatalf("collection %q, selected %q; want the config's tree opened on its first request", app.col.Name, selectedName(app))
	}
}

func TestStartWithAnEmptyClientConfigKeepsTheSample(t *testing.T) {
	app, err := NewApp(Options{Sample: collection.GitHubSample, Clients: fakeClients{}})
	if err != nil {
		t.Fatal(err)
	}
	if app.col.Name != collection.GitHubSample().Name || selectedName(app) != "Search Repos" {
		t.Fatalf("collection %q, selected %q; want the sample on its search call", app.col.Name, selectedName(app))
	}
}

func TestStartWithABrokenClientConfigSaysSoAndKeepsTheSample(t *testing.T) {
	app, err := NewApp(Options{Sample: collection.GitHubSample, Clients: fakeClients{err: errors.New("parse .workshop.yaml: line 3")}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(app.status, "Client config not loaded") || !strings.Contains(app.status, ".workshop.yaml") {
		t.Fatalf("status = %q, want the problem and the file", app.status)
	}
	if app.col.Name != collection.GitHubSample().Name {
		t.Fatalf("collection %q, want the sample kept", app.col.Name)
	}
}

func TestStartWithAnEmptySampleOpensANewRequest(t *testing.T) {
	empty := func() model.Collection { return model.Collection{Folder: model.Folder{Name: "Workshop"}} }
	app, err := NewApp(Options{Sample: empty})
	if err != nil {
		t.Fatal(err)
	}
	if selectedName(app) != "New Request" {
		t.Fatalf("selected = %q, want a new request in the empty collection", selectedName(app))
	}
}

func TestStartWithoutASampleOptionUsesTheGitHubSample(t *testing.T) {
	app, err := NewApp(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if app.col.Name != collection.GitHubSample().Name || len(app.col.Folders) != 3 {
		t.Fatalf("collection %q with %d folders, want the GitHub sample", app.col.Name, len(app.col.Folders))
	}
}

func selectedName(app *App) string {
	request, ok := app.selected.(*model.Request)
	if !ok {
		return ""
	}
	return request.Name
}
