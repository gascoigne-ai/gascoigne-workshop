package collection

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

func TestSampleStartIsTheSearchCallOfTheGitHubSample(t *testing.T) {
	sample := GitHubSample()
	folder, call, ok := SampleStart(&sample)
	if !ok || folder != sample.Folders[1] || call.Name != "Search Repos" {
		t.Fatalf("folder %v request %v ok %v", folder, call, ok)
	}
}

func TestSampleStartIsMissingFromACollectionWithoutASecondFolderWithRequests(t *testing.T) {
	one := request.Collection{Folder: request.Folder{Folders: []*request.Folder{{Name: "only"}}}}
	empty := request.Collection{Folder: request.Folder{Folders: []*request.Folder{{}, {Name: "no requests"}}}}
	for name, col := range map[string]request.Collection{"one folder": one, "empty second folder": empty} {
		t.Run(name, func(t *testing.T) {
			if _, _, ok := SampleStart(&col); ok {
				t.Fatal("ok, want none")
			}
		})
	}
}
