package model

import (
	"reflect"
	"testing"
)

func TestAttachKeepsPointerAndRenames(t *testing.T) {
	cases := []struct {
		name        string
		node        func(root *Folder) Node
		into        func(root *Folder) *Folder
		before      func(root *Folder) Node
		wantName    string
		wantRenamed bool
		wantOrder   func(into *Folder) []string
	}{
		{
			name:     "no clash keeps the name",
			node:     func(*Folder) Node { return Of(&Request{Name: "new"}) },
			into:     func(root *Folder) *Folder { return root.Folders[1] },
			wantName: "new",
			wantOrder: func(into *Folder) []string {
				return names(into.Requests)
			},
		},
		{
			name:        "request clash renames",
			node:        func(*Folder) Node { return Of(&Request{Name: "b1"}) },
			into:        func(root *Folder) *Folder { return root.Folders[1] },
			wantName:    "b1 2",
			wantRenamed: true,
		},
		{
			name:        "folder name clashes with a request",
			node:        func(*Folder) Node { return Of(&Folder{Name: "b1"}) },
			into:        func(root *Folder) *Folder { return root.Folders[1] },
			wantName:    "b1 2",
			wantRenamed: true,
		},
		{
			name:     "before a sibling",
			node:     func(*Folder) Node { return Of(&Request{Name: "new"}) },
			into:     func(root *Folder) *Folder { return root.Folders[0] },
			before:   func(root *Folder) Node { return Of(root.Folders[0].Requests[1]) },
			wantName: "new",
			wantOrder: func(into *Folder) []string {
				return names(into.Requests)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, _, _, _, _, _, _, _, _ := fixture()
			node, into := tc.node(root), tc.into(root)
			var before Node
			if tc.before != nil {
				before = tc.before(root)
			}
			renamed := into.Attach(node, before)
			if renamed != tc.wantRenamed || node.Name() != tc.wantName {
				t.Fatalf("renamed=%v name=%q, want %v %q", renamed, node.Name(), tc.wantRenamed, tc.wantName)
			}
			if !into.Contains(node) {
				t.Fatalf("the node pointer is not in the folder")
			}
			if tc.wantOrder == nil {
				return
			}
			want := map[string][]string{
				"no clash keeps the name": {"b1", "new"},
				"before a sibling":        {"a1", "new", "a2"},
			}[tc.name]
			if got := tc.wantOrder(into); !reflect.DeepEqual(got, want) {
				t.Fatalf("order = %v, want %v", got, want)
			}
		})
	}
}

func TestAttachSameFolderReorderKeepsName(t *testing.T) {
	root, a, _, _, a1, a2, _, _, _ := fixture()
	if !root.Delete(Of(a2)) {
		t.Fatal("detach failed")
	}
	if renamed := a.Attach(Of(a2), Of(a1)); renamed || a2.Name != "a2" {
		t.Fatalf("renamed=%v name=%q, want a reorder that keeps the name", renamed, a2.Name)
	}
	if got := names(a.Requests); !reflect.DeepEqual(got, []string{"a2", "a1"}) {
		t.Fatalf("order = %v", got)
	}
}

func TestNodeCloneIsDeep(t *testing.T) {
	root, a, _, _, a1, _, x, _, _ := fixture()
	a1.Params = []KV{{Name: "p", Enabled: true}}
	a.Vars = []KV{{Name: "v", Value: "1", Enabled: true}}

	reqClone := Of(a1).Clone()
	if reqClone.Req == a1 || !reflect.DeepEqual(*reqClone.Req, *a1) {
		t.Fatalf("request clone must be a new pointer with equal content")
	}
	reqClone.Req.Params[0].Name = "changed"
	if a1.Params[0].Name != "p" {
		t.Fatal("request clone shares rows with the original")
	}

	dirClone := Of(a).Clone()
	if dirClone.Dir == a || root.Contains(dirClone) {
		t.Fatal("folder clone must be a new pointer outside the tree")
	}
	dirClone.Dir.Vars[0].Value = "2"
	dirClone.Dir.Folders[0].Requests[0].Name = "changed"
	if a.Vars[0].Value != "1" || x.Name != "x" {
		t.Fatal("folder clone shares vars or children with the original")
	}
	if !Of(nil).Clone().None() {
		t.Fatal("clone of none is none")
	}
}

func TestUsedVarsEnabledRowsOnlyInFirstUseOrder(t *testing.T) {
	request := &Request{
		URL:         "{{host}}/{{path}}",
		Params:      []KV{{Name: "q", Value: "{{query}}", Enabled: true}, {Name: "off", Value: "{{skipped}}"}},
		Headers:     []KV{{Name: "X-{{hdr}}", Value: "{{host}}", Enabled: true}},
		Form:        []KV{{Name: "f", Value: "{{field}}", Enabled: true}},
		Body:        `{"a":"{{bodyvar}}"}`,
		GraphQLVars: `{"id":"{{gqlvar}}"}`,
		Auth:        Auth{Token: "{{token}}", User: "{{user}}", Password: "{{pass}}", KeyName: "{{kn}}", KeyValue: "{{kv}}"},
	}
	want := []string{"host", "path", "query", "hdr", "field", "bodyvar", "gqlvar", "token", "user", "pass", "kn", "kv"}
	if got := UsedVars(request); !reflect.DeepEqual(got, want) {
		t.Fatalf("UsedVars = %v, want %v", got, want)
	}
	if got := UsedVars(&Request{}); len(got) != 0 {
		t.Fatalf("UsedVars of an empty request = %v", got)
	}
}
