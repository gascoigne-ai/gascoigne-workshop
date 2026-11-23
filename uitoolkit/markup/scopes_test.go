package markup

import (
	"maps"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// buildWith parses src and builds it with the scopes.
func buildWith(t *testing.T, src string, own Scope, fallbacks ...Scope) (view.Node, error) {
	t.Helper()
	template, err := Parse("t.vml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return template.Build(own, fallbacks...)
}

// The name a template reads comes from the loop variable first, then its own
// scope, then each fallback in order.
func TestBuildReadsFallbackScopesInOrder(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		own       Scope
		fallbacks []Scope
		want      string // a substring of the dump
		wantErr   string
	}{
		{"own wins", `<text>{{a}}</text>`, Scope{"a": "own"}, []Scope{{"a": "one"}, {"a": "two"}}, `text="own"`, ""},
		{"first fallback beats a later one", `<text>{{a}}</text>`, Scope{}, []Scope{{"a": "one"}, {"a": "two"}}, `text="one"`, ""},
		{"a later fallback is read", `<text>{{a}}</text>`, Scope{}, []Scope{{"b": "one"}, {"a": "two"}}, `text="two"`, ""},
		{"a nil entry shadows a fallback", `<text if="a">x</text>`, Scope{"a": nil}, []Scope{{"a": true}}, "", ""},
		{"a loop variable beats all", `<row><text for="a in xs">{{a}}</text></row>`, Scope{"xs": []any{"loop"}, "a": "own"}, []Scope{{"a": "one"}}, `text="loop"`, ""},
		{"a nil loop item shadows", `<row><text for="a in xs" if="a">x</text></row>`, Scope{"xs": []any{nil}}, []Scope{{"a": true}}, "", ""},
		{"$index is not read from a fallback", `<row><text for="a in xs">{{$index}}</text></row>`, Scope{"xs": []any{"x"}}, []Scope{{"$index": "fallback"}}, `text="0"`, ""},
		{"a dotted path reads a fallback", `<text>{{a.b}}</text>`, Scope{}, []Scope{{"a": map[string]any{"b": "deep"}}}, `text="deep"`, ""},
		{"an own entry hides the whole fallback entry", `<text>{{a.b}}</text>`, Scope{"a": map[string]any{}}, []Scope{{"a": map[string]any{"b": "deep"}}}, "", `unknown name "a.b"`},
		{"an unknown name reports its full path", `<text>{{a.b}}</text>`, Scope{}, []Scope{{"c": 1}}, "", `unknown name "a.b"`},
		{"a handler is read from a fallback", `<row on-click="act"/>`, Scope{}, []Scope{{"act": func() {}}}, "", ""},
		{"a param is read from a fallback", `<scroll state="s"><text>x</text></scroll>`, Scope{}, []Scope{{"s": &widget.ScrollState{}}}, "", ""},
		{"an if reads a fallback", `<row><text if="!a">x</text></row>`, Scope{}, []Scope{{"a": false}}, `text="x"`, ""},
		{"a for reads a fallback list", `<row><text for="x in xs">{{x}}</text></row>`, Scope{}, []Scope{{"xs": []any{"f"}}}, `text="f"`, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node, err := buildWith(t, test.src, test.own, test.fallbacks...)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if dump := view.Dump(node); !strings.Contains(dump, test.want) {
				t.Errorf("dump lacks %q:\n%s", test.want, dump)
			}
		})
	}
}

// A build reads its scopes and writes none of them: not the loop variable,
// not $index.
func TestBuildLeavesItsScopesUnchanged(t *testing.T) {
	own := Scope{"xs": []any{"a", "b"}, "x": "own", "$index": "own"}
	fallback := Scope{"y": "f"}
	ownBefore, fallbackBefore := maps.Clone(own), maps.Clone(fallback)
	if _, err := buildWith(t, `<row><text for="x in xs">{{$index}}{{x}}{{y}}</text></row>`, own, fallback); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(own, ownBefore) || !reflect.DeepEqual(fallback, fallbackBefore) {
		t.Errorf("scopes changed: own %v, fallback %v", own, fallback)
	}
}

type loopItem struct {
	Name string
	Open func()
}

// Each item's handler is its own: a binding reused for the next item must not
// leak into a handler built for the one before.
func TestForHandlersKeepTheirOwnItem(t *testing.T) {
	var clicked []string
	item := func(name string) loopItem {
		return loopItem{Name: name, Open: func() { clicked = append(clicked, name) }}
	}
	node, err := buildWith(t, `<col><row for="x in xs" h="20" on-click="x.Open"><text>{{x.Name}}</text></row></col>`,
		Scope{"xs": []loopItem{item("a"), item("b"), item("c")}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	view.Layout(ctx, node, 100, 100)
	for _, posY := range []int{50, 10, 30} {
		ctx.In = core.Input{X: 5, Y: posY, Click: true}
		view.Update(ctx, node)
	}
	if got, want := strings.Join(clicked, ""), "cab"; got != want {
		t.Errorf("clicks reached %q, want %q", got, want)
	}
}

// An inner loop's $index shadows the outer one while it runs, and the outer
// one is back after.
func TestNestedForIndexShadowsOuterIndex(t *testing.T) {
	node, err := buildWith(t, `<col><col for="a in rows"><text>{{$index}}</text><text for="b in a">{{$index}}</text><text>{{$index}}</text></col></col>`,
		Scope{"rows": []any{[]any{"p", "q"}, []any{"r"}}})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, line := range strings.Split(view.Dump(node), "\n") {
		if _, after, found := strings.Cut(line, `text="`); found {
			got = append(got, after[:strings.Index(after, `"`)])
		}
	}
	if want := "0 0 1 0 1 0 1"; strings.Join(got, " ") != want {
		t.Errorf("texts = %q, want %q", strings.Join(got, " "), want)
	}
}

func TestLoopVariableShadowsScopeEntry(t *testing.T) {
	node, err := buildWith(t, `<row><text for="x in xs">{{x}}</text><text>{{x}}</text></row>`, Scope{"xs": []any{"in"}, "x": "out"})
	if err != nil {
		t.Fatal(err)
	}
	dump := view.Dump(node)
	if !strings.Contains(dump, `text="in"`) || !strings.Contains(dump, `text="out"`) {
		t.Errorf("want the loop value inside and the scope value outside:\n%s", dump)
	}
}

// The error text of a failed read names the full dotted path, from a loop
// item as from a scope.
func TestBuildErrorTextsNameTheFullPath(t *testing.T) {
	tests := []struct {
		name, src string
		own       Scope
		want      string
	}{
		{"scope", `<text>{{a.b.c}}</text>`, Scope{"a": map[string]any{}}, `t.vml:1: unknown name "a.b.c"`},
		{"loop item", `<row><text for="x in xs">{{x.nope}}</text></row>`, Scope{"xs": []any{map[string]any{}}}, `unknown name "x.nope"`},
		{"loop item not an object", `<row><text for="x in xs">{{x.nope}}</text></row>`, Scope{"xs": []any{3}}, `"x.nope": x is not an object`},
		{"nil loop item", `<row><text for="x in xs">{{x.nope}}</text></row>`, Scope{"xs": []any{nil}}, `"x.nope": x is not an object`},
		{"for list", `<row><text for="x in nope">a</text></row>`, Scope{}, `unknown name "nope"`},
		{"for not a list", `<row><text for="x in n">a</text></row>`, Scope{"n": 3}, `n: is not a list`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildWith(t, test.src, test.own)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("error = %v, want it to hold %q", err, test.want)
			}
		})
	}
}
