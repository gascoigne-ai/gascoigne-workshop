package collection

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestSampleStartIsTheSearchCallOfTheGitHubSample(t *testing.T) {
	sample := GitHubSample()
	folder, request, ok := SampleStart(&sample)
	if !ok || folder != sample.Folders[1] || request.Name != "Search Repos" {
		t.Fatalf("folder %v request %v ok %v", folder, request, ok)
	}
}

func TestSampleStartIsMissingFromACollectionWithoutASecondFolderWithRequests(t *testing.T) {
	one := model.Collection{Folder: model.Folder{Folders: []*model.Folder{{Name: "only"}}}}
	empty := model.Collection{Folder: model.Folder{Folders: []*model.Folder{{}, {Name: "no requests"}}}}
	for name, col := range map[string]model.Collection{"one folder": one, "empty second folder": empty} {
		t.Run(name, func(t *testing.T) {
			if _, _, ok := SampleStart(&col); ok {
				t.Fatal("ok, want none")
			}
		})
	}
}
