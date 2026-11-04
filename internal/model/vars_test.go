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
