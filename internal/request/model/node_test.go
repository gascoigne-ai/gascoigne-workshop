package model

import "testing"

func TestOfWrapsARequestOrAFolderAndNothingElse(t *testing.T) {
	request, folder := &Request{Name: "r"}, &Folder{Name: "f"}
	cases := []struct {
		name string
		node any
		want Node
	}{
		{"a request", request, Node{Req: request}},
		{"a folder", folder, Node{Dir: folder}},
		{"nil", nil, Node{}},
		{"something else", 42, Node{}},
		{"a nil request", (*Request)(nil), Node{}},
		{"a nil folder", (*Folder)(nil), Node{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Of(tc.node); got != tc.want {
				t.Errorf("Of = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestANodeKnowsItsNameAndWhetherItIsNone(t *testing.T) {
	request, folder := &Request{Name: "r"}, &Folder{Name: "f"}
	cases := []struct {
		name     string
		node     Node
		wantName string
		wantNone bool
	}{
		{"request", Of(request), "r", false},
		{"folder", Of(folder), "f", false},
		{"none", Node{}, "", true},
		{"typed nil", Of((*Folder)(nil)), "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.node.Name() != tc.wantName || tc.node.None() != tc.wantNone {
				t.Errorf("name %q none %v, want %q %v", tc.node.Name(), tc.node.None(), tc.wantName, tc.wantNone)
			}
		})
	}
}
