package collections

import (
	"strings"
	"testing"
)

func TestCleanNameTrimsAndRejects(t *testing.T) {
	cases := []struct {
		in, want string
		fails    bool
	}{
		{"  Echo  ", "Echo", false},
		{"Ünï cødé", "Ünï cødé", false},
		{"", "", true},
		{"   ", "", true},
		{strings.Repeat("a", MaxNameLen), strings.Repeat("a", MaxNameLen), false},
		{strings.Repeat("a", MaxNameLen+1), "", true},
		{strings.Repeat("é", MaxNameLen), strings.Repeat("é", MaxNameLen), false}, // characters, not bytes
	}
	for _, tc := range cases {
		got, err := CleanName(tc.in)
		if (err != nil) != tc.fails || got != tc.want {
			t.Errorf("CleanName(%q) = %q, %v; want %q, fails %v", tc.in, got, err, tc.want, tc.fails)
		}
	}
}

func TestDeletePromptNamesTheCollection(t *testing.T) {
	title, body := DeletePrompt("Echo")
	if title != "Delete Echo?" || !strings.Contains(body, "cannot be undone") {
		t.Fatalf("got %q, %q", title, body)
	}
}

func TestClosePromptNamesTheCollectionAndWhatIsLost(t *testing.T) {
	title, body := ClosePrompt("Echo")
	if title != "Close Echo?" || !strings.Contains(body, "changes are lost") {
		t.Fatalf("got %q, %q", title, body)
	}
}

func TestNextAfterDelete(t *testing.T) {
	cases := []struct {
		name    string
		dirs    []string
		deleted string
		want    string
	}{
		{"the one after", []string{"a", "b", "c"}, "b", "c"},
		{"the first has a successor", []string{"a", "b"}, "a", "b"},
		{"the last falls back to the one before", []string{"a", "b", "c"}, "c", "b"},
		{"the only one leaves none", []string{"a"}, "a", ""},
		{"not listed any more", []string{"a", "c"}, "b", "c"},
		{"nothing listed", nil, "b", ""},
	}
	for _, tc := range cases {
		if got := NextAfterDelete(tc.dirs, tc.deleted); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
