package markup

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// withCountingAttribute adds an attribute "counted" that counts how often its
// option is made, and removes it when the test ends.
func withCountingAttribute(t *testing.T) *int {
	t.Helper()
	made := 0
	attributes["counted"] = func(string) (view.Option, error) {
		made++
		return view.W(7), nil
	}
	t.Cleanup(func() { delete(attributes, "counted") })
	return &made
}

func TestStaticAttributesAreBuiltOnce(t *testing.T) {
	made := withCountingAttribute(t)
	template, err := Parse("t.vml", []byte(`<row counted="1"/>`))
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		node, err := template.Build(Scope{})
		if err != nil {
			t.Fatal(err)
		}
		if got := view.Dump(node); got == "" {
			t.Fatal("empty dump")
		}
	}
	if *made != 1 {
		t.Errorf("the option of a static attribute was made %d times in a parse and 3 builds, want 1", *made)
	}
}

func TestDynamicAttributesAreBuiltEveryTime(t *testing.T) {
	made := withCountingAttribute(t)
	template, err := Parse("t.vml", []byte(`<row counted="{{v}}"/>`))
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := template.Build(Scope{"v": 1}); err != nil {
			t.Fatal(err)
		}
	}
	if *made != 3 {
		t.Errorf("the option of a dynamic attribute was made %d times in 3 builds, want 3", *made)
	}
}

// A shared option must give the same element every time.
func TestStaticOptionsApplyToEveryBuild(t *testing.T) {
	template := Must("t.vml", `<row pad="3 4" w="40" bg="panel" small><text bold>x</text></row>`)
	first := view.Dump(template.MustBuild(Scope{}))
	second := view.Dump(template.MustBuild(Scope{}))
	if first != second {
		t.Fatalf("builds differ:\n%s\n%s", first, second)
	}
}

// A static attribute with a bad value fails at Build, with its line, and
// not at all while an earlier `if` hides the element (as before options
// were made at Parse).
func TestBadStaticAttributesFailAtBuildWithTheirLine(t *testing.T) {
	for name, tt := range map[string]struct {
		src, want string
		scope     Scope
	}{
		"number":         {"<row>\n<row w=\"wide\"/></row>", `t.vml:2: w: "wide" is not a whole number`, Scope{}},
		"pad":            {"<row\n pad=\"1 2 3\"/>", `t.vml:2: pad: pad takes one or two numbers, not "1 2 3"`, Scope{}},
		"flag":           {`<text small="maybe">a</text>`, `t.vml:1: small: "maybe" is not true or false`, Scope{}},
		"after if shown": {`<row if="on" w="wide"/>`, `t.vml:1: w: "wide" is not a whole number`, Scope{"on": true}},
		"hidden by if":   {`<row if="on" w="wide"/>`, "", Scope{"on": false}},
	} {
		template, err := Parse("t.vml", []byte(tt.src))
		if err != nil {
			t.Errorf("%s: parse: %v", name, err)
			continue
		}
		_, err = template.Build(tt.scope)
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("%s: err = %v, want none", name, err)
		case tt.want != "" && (err == nil || err.Error() != tt.want):
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestPathsAreSplitAtParse(t *testing.T) {
	template, err := Parse("t.vml", []byte(`<text bg="x {{a.b}}">{{c.d}} and {{e}}</text>`))
	if err != nil {
		t.Fatal(err)
	}
	want := func(parts interp, paths ...[]string) {
		t.Helper()
		var got [][]string
		for _, part := range parts.parts {
			if part.path != nil {
				got = append(got, part.path)
			}
		}
		if len(got) != len(paths) {
			t.Fatalf("paths = %v, want %v", got, paths)
		}
		for i := range paths {
			if len(got[i]) != len(paths[i]) {
				t.Fatalf("paths = %v, want %v", got, paths)
			}
			for j := range paths[i] {
				if got[i][j] != paths[i][j] {
					t.Fatalf("paths = %v, want %v", got, paths)
				}
			}
		}
	}
	want(template.root.textParts, []string{"c", "d"}, []string{"e"})
	want(template.root.attrs[0].parts, []string{"a", "b"})
}
