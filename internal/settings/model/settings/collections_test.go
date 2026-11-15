package settings

import (
	"encoding/json"
	"reflect"
	"testing"
)

// A settings file written when one collection was open at a time keeps
// working: its workspace and environment are read as a set of one.
func TestRememberedCollections(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		want     []OpenCollection
	}{
		{"nothing remembered", Settings{}, nil},
		{"the open set", Settings{Collections: []OpenCollection{{Folder: "/a", Environment: "Dev"}, {Folder: "/b"}}},
			[]OpenCollection{{Folder: "/a", Environment: "Dev"}, {Folder: "/b"}}},
		{"an older file's workspace and environment", Settings{Workspace: "/w", Environment: "Dev"},
			[]OpenCollection{{Folder: "/w", Environment: "Dev"}}},
		{"an older file's environment of the built-in collection", Settings{Environment: "Dev"},
			[]OpenCollection{{Environment: "Dev"}}},
		{"the open set wins over the older keys", Settings{Collections: []OpenCollection{{Folder: "/a"}}, Workspace: "/w", Environment: "Dev"},
			[]OpenCollection{{Folder: "/a"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.settings.RememberedCollections(); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestRememberCollectionsWritesOnlyTheNewKey(t *testing.T) {
	s := Default()
	s.Workspace, s.Environment = "/w", "Dev"
	s.RememberCollections([]OpenCollection{{Folder: "/w", Environment: "Dev"}, {Folder: "/x"}})
	text, err := Minimal(s)
	if err != nil {
		t.Fatal(err)
	}
	var written struct {
		General map[string]json.RawMessage `json:"general"`
	}
	if err := json.Unmarshal(text, &written); err != nil {
		t.Fatal(err)
	}
	if len(written.General) != 1 || written.General["collections"] == nil {
		t.Fatalf("general section %s, want collections only", text)
	}
	back, err := Decode(text)
	if err != nil || !reflect.DeepEqual(back.RememberedCollections(), []OpenCollection{{Folder: "/w", Environment: "Dev"}, {Folder: "/x"}}) {
		t.Fatalf("read back %+v %v", back.Collections, err)
	}
}
