package model

import (
	"reflect"
	"testing"
)

func TestMergeVarsLetsEnabledOverridesWin(t *testing.T) {
	base := []KV{{Name: "host", Value: "prod", Enabled: true}, {Name: "key", Value: "k", Enabled: true}}
	over := []KV{{Name: "host", Value: "dev", Enabled: true}, {Name: "key", Value: "off", Enabled: false}}
	got := MergeVars(base, over)
	if Expand("{{host}} {{key}}", got) != "dev k" {
		t.Fatalf("%+v -> %q", got, Expand("{{host}} {{key}}", got))
	}
	if len(MergeVars(nil, nil)) != 0 {
		t.Fatal("empty merge not empty")
	}
}

func TestUnresolved(t *testing.T) {
	vars := []KV{{Name: "a", Value: "1", Enabled: true}, {Name: "off", Value: "x"}}
	got := Unresolved("{{a}}/{{b}}/{{off}}/{{b}}/{{", vars)
	if want := []string{"b", "off"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestVarsForFollowsTheResolutionOrder(t *testing.T) {
	kv := func(n, v string) KV { return KV{Name: n, Value: v, Enabled: true} }
	req := &Request{Name: "r"}
	inner := &Folder{Name: "inner", Vars: []KV{kv("who", "inner")}, Requests: []*Request{req}}
	outer := &Folder{Name: "outer", Vars: []KV{kv("who", "outer"), kv("only-outer", "o")}, Folders: []*Folder{inner}}
	c := &Collection{Folder: Folder{Name: "c", Folders: []*Folder{outer}}, Vars: []KV{kv("who", "collection"), kv("env", "collection"), kv("only-collection", "c")}}
	dev := &Environment{Name: "Dev", Vars: []KV{kv("env", "dev"), kv("who", "dev")}}

	got := func(env *Environment) string {
		return Expand("{{who}} {{env}} {{only-outer}} {{only-collection}}", c.VarsFor(req, env))
	}
	if g := got(nil); g != "inner collection o c" {
		t.Fatalf("no environment: %q", g)
	}
	if g := got(dev); g != "inner dev o c" {
		t.Fatalf("environment under the folders: %q", g)
	}
	inner.Vars = nil
	if g := got(dev); g != "outer dev o c" {
		t.Fatalf("outer folder wins over the environment: %q", g)
	}
	outer.Vars = nil
	if g := got(dev); g != "dev dev {{only-outer}} c" {
		t.Fatalf("environment wins over the collection: %q", g)
	}
	root := &Request{Name: "at the root"}
	c.Requests = append(c.Requests, root)
	if g := Expand("{{who}}", c.VarsFor(root, nil)); g != "collection" {
		t.Fatalf("a request at the root sees the collection's variables: %q", g)
	}
}

func TestVarCompletionFiltersByWhatFollowsAnOpenBrace(t *testing.T) {
	vars := []KV{{Name: "host", Enabled: true}, {Name: "hostname", Enabled: true}, {Name: "token", Enabled: true}, {Name: "off"}, {Name: "host", Enabled: true}}
	names, from := VarCompletion("https://{{ho", 12, vars)
	if from != 8 || !reflect.DeepEqual(names, []string{"host", "hostname"}) {
		t.Fatalf("%v from %d", names, from)
	}
	names, from = VarCompletion("https://x/", 10, vars)
	if from != -1 || !reflect.DeepEqual(names, []string{"host", "hostname", "token"}) {
		t.Fatalf("outside braces: %v from %d", names, from)
	}
	if names, from = VarCompletion("{{host}} {{", 11, vars); from != 9 || len(names) != 3 {
		t.Fatalf("a closed brace pair ends the partial name: %v %d", names, from)
	}
	if names, from = VarCompletion("ż{{t", 99, vars); from != 1 || len(names) != 1 || names[0] != "token" {
		t.Fatalf("runes, and a cursor past the end: %v %d", names, from)
	}
}

func TestUpsertVarUpdatesInPlaceOrAppends(t *testing.T) {
	rows := []KV{{Name: "a", Value: "1"}}
	rows = UpsertVar(rows, "a", "2")
	rows = UpsertVar(rows, "b", "3")
	if len(rows) != 2 || rows[0].Value != "2" || !rows[0].Enabled || rows[1].Name != "b" {
		t.Fatalf("%+v", rows)
	}
}
