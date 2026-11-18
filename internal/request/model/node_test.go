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
		wantAny  any
	}{
		{"request", Of(request), "r", false, request},
		{"folder", Of(folder), "f", false, folder},
		{"none", Node{}, "", true, nil},
		{"typed nil", Of((*Folder)(nil)), "", true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.node.Name() != tc.wantName || tc.node.None() != tc.wantNone || tc.node.Any() != tc.wantAny {
				t.Errorf("name %q none %v any %v, want %q %v %v", tc.node.Name(), tc.node.None(), tc.node.Any(), tc.wantName, tc.wantNone, tc.wantAny)
			}
		})
	}
}
