package component

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

func build(t *testing.T, src string, sc Scope) vml.Node {
	t.Helper()
	tp, err := Parse("t.vml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	n, err := tp.Build(sc)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// The same view written in Go and as a template dumps identically, laid out.
func TestTemplateMatchesTheGoDSL(t *testing.T) {
	c, err := core.NewCtx(core.Dark())
	if err != nil {
		t.Fatal(err)
	}
	clicked := false
	goView := vml.Row(vml.H(28), vml.PadX(12), vml.Gap(12), vml.Bg(core.Frame), vml.Control("statusbar"), vml.BorderT(core.Border),
		vml.Text("Ready", vml.Small(), vml.Muted(), vml.Ellipsis(), vml.Grow(1)),
		vml.Text("v0.1.0", vml.Small(), vml.Muted(), vml.OnClick(func() { clicked = true })),
		vml.Button("Go", vml.W(60), vml.Tooltip("Go on")),
	)
	tplView := build(t, `
		<row h="28" pad="12 0" gap="12" bg="frame" control="statusbar" border-t="border">
		  <text small muted ellipsis grow="1">{{status}}</text>
		  <text small muted onclick="bump">{{version}}</text>
		  <button w="60" tip="Go on">Go</button>
		</row>`, Scope{"status": "Ready", "version": "v0.1.0", "bump": func() { clicked = true }})
	vml.Layout(c, goView, 600, 28)
	vml.Layout(c, tplView, 600, 28)
	if g, w := vml.Dump(tplView), vml.Dump(goView); g != w {
		t.Fatalf("template:\n%s\nGo:\n%s", g, w)
	}
	_ = clicked
}

func TestIfForAndInterpolation(t *testing.T) {
	src := `<col>
	  <text if="show">shown {{n}}</text>
	  <text if="!show">hidden</text>
	  <text for="it in items">{{$index}}:{{it.name}}</text>
	</col>`
	on := vml.Dump(build(t, src, Scope{"show": true, "n": 3, "items": []map[string]any{{"name": "a"}, {"name": "b"}}}))
	for _, want := range []string{`text="shown 3"`, `text="0:a"`, `text="1:b"`} {
		if !strings.Contains(on, want) {
			t.Errorf("missing %s in\n%s", want, on)
		}
	}
	if strings.Contains(on, "hidden") {
		t.Errorf("if=!show built when show is true:\n%s", on)
	}
	off := vml.Dump(build(t, src, Scope{"show": false, "n": 0, "items": []string{}}))
	if strings.Contains(off, "shown") || !strings.Contains(off, `text="hidden"`) || strings.Contains(off, "0:") {
		t.Errorf("show=false:\n%s", off)
	}
}

func TestAttributesCanBeInterpolated(t *testing.T) {
	n := build(t, `<row bg="{{tone}}" w="{{width}}"><text color="{{tone}}">x</text></row>`, Scope{"tone": "panel", "width": 40})
	d := vml.Dump(n)
	if !strings.Contains(d, "bg=panel") || !strings.Contains(d, "w=40") || !strings.Contains(d, "fg=panel") {
		t.Fatal(d)
	}
}

func TestParseErrorsNameTheFileAndLine(t *testing.T) {
	for name, tt := range map[string]struct{ src, want string }{
		"unknown element":   {"<row>\n  <blink/>\n</row>", "t.vml:2: unknown element <blink>"},
		"unknown attribute": {"<row>\n\n<text sparkle=\"1\">x</text></row>", "t.vml:3: unknown attribute \"sparkle\""},
		"text in a row":     {"<row>\n loose text\n</row>", "text is only allowed inside"},
		"two roots":         {"<row></row>\n<row></row>", "one root element"},
		"leaf with child":   {"<text><row/></text>", "cannot contain"},
		"empty":             {"  ", "empty"},
		"unclosed":          {"<row><text>x</text>", "unexpected EOF"},
	} {
		_, err := Parse("t.vml", []byte(tt.src))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestBuildErrorsNameTheFileAndLine(t *testing.T) {
	for name, tt := range map[string]struct {
		src  string
		sc   Scope
		want string
	}{
		"unknown name":  {"<row>\n<text>{{nope}}</text></row>", Scope{}, `t.vml:2: unknown name "nope"`},
		"bad number":    {`<row w="wide"/>`, Scope{}, "not a whole number"},
		"bad pad":       {`<row pad="1 2 3"/>`, Scope{}, "pad takes one or two"},
		"not a handler": {`<text onclick="x">a</text>`, Scope{"x": "str"}, "not a func()"},
		"bad for":       {`<col><text for="items">a</text></col>`, Scope{}, `must read "item in list"`},
		"not a list":    {`<col><text for="i in x">a</text></col>`, Scope{"x": 3}, "is not a list"},
		"bad flag":      {`<text small="maybe">a</text>`, Scope{}, "not true or false"},
		"open braces":   {`<text>{{a</text>`, Scope{}, "without"},
		"dotted miss":   {`<text>{{a.b}}</text>`, Scope{"a": map[string]any{}}, `unknown name "a.b"`},
		"dotted scalar": {`<text>{{a.b}}</text>`, Scope{"a": 1}, "is not an object"},
	} {
		tp, err := Parse("t.vml", []byte(tt.src))
		if err != nil {
			t.Errorf("%s: parse: %v", name, err)
			continue
		}
		if _, err := tp.Build(tt.sc); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestFlagsAcceptFalseAndHTMLEntities(t *testing.T) {
	d := vml.Dump(build(t, `<text small="false" bold="true">a&nbsp;b &amp; c</text>`, nil))
	if strings.Contains(d, "small=true") || !strings.Contains(d, "bold=true") || !strings.Contains(d, "&") {
		t.Fatal(d)
	}
}

func TestRootIfFalseBuildsNothing(t *testing.T) {
	tp, err := Parse("t.vml", []byte(`<row if="on"/>`))
	if err != nil {
		t.Fatal(err)
	}
	n, err := tp.Build(Scope{"on": false})
	if err != nil || n != nil {
		t.Fatalf("%v %v", n, err)
	}
}

// An inline template, written in Go source, builds the same tree as a file.
func TestInlineTemplateWithMust(t *testing.T) {
	bar := Must("inline", `<row h="28"><text small>{{status}}</text></row>`)
	got := vml.Dump(bar.MustBuild(Scope{"status": "Ready"}))
	want := vml.Dump(vml.Row(vml.H(28), vml.Text("Ready", vml.Small())))
	if got != want {
		t.Fatalf("inline:\n%s\nGo:\n%s", got, want)
	}
	if bar.Name() != "inline" {
		t.Fatalf("name %q", bar.Name())
	}
	for name, f := range map[string]func(){
		"bad syntax": func() { Must("bad", `<row><blink/></row>`) },
		"bad scope":  func() { bar.MustBuild(Scope{}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: no panic", name)
				}
			}()
			f()
		}()
	}
}

func TestViewParsesEachSourceOnce(t *testing.T) {
	const src = `<text>{{n}}</text>`
	a := View(src, Scope{"n": 1})
	b := View(src, Scope{"n": 2})
	if !strings.Contains(vml.Dump(a), `"1"`) || !strings.Contains(vml.Dump(b), `"2"`) {
		t.Fatal("scope not applied per call")
	}
	viewMu.Lock()
	n := len(viewCache)
	_, ok := viewCache[src]
	viewMu.Unlock()
	if !ok || n < 1 {
		t.Fatal("source not cached")
	}
}

func TestErrorsSuggestAndListTheValidNames(t *testing.T) {
	_, err := Parse("t.vml", []byte(`<row><txt/></row>`))
	if err == nil || !strings.Contains(err.Error(), `did you mean "text"`) || !strings.Contains(err.Error(), "elements:") {
		t.Fatalf("%v", err)
	}
	_, err = Parse("t.vml", []byte(`<row border_top="border"/>`))
	if err == nil || !strings.Contains(err.Error(), `did you mean "border-t"`) {
		t.Fatalf("%v", err)
	}
	_, err = Parse("t.vml", []byte(`<row zzzzzzzz="1"/>`))
	if err == nil || strings.Contains(err.Error(), "did you mean") || !strings.Contains(err.Error(), "attributes:") {
		t.Fatalf("a far-off name must not be guessed: %v", err)
	}
}

func TestReferenceListsEveryElementAndAttribute(t *testing.T) {
	ref := Reference()
	for n := range elements {
		if !strings.Contains(ref, "<"+n+">") {
			t.Errorf("reference misses element %s", n)
		}
	}
	for n := range attributes {
		if !strings.Contains(ref, "  "+n+"\n") {
			t.Errorf("reference misses attribute %s", n)
		}
	}
	for n := range directives {
		if !strings.Contains(ref, "  "+n+"\n") {
			t.Errorf("reference misses directive %s", n)
		}
	}
}

// Template attributes are the Go options in kebab-case, for every side.
func TestSideAttributesMatchTheGoOptions(t *testing.T) {
	got := vml.Dump(build(t, `<row border-r="a" border-x="b" border-y="c" margin-l="1" margin-y="2" pad-b="3"/>`, nil))
	want := vml.Dump(vml.Row(vml.BorderR("a"), vml.BorderX("b"), vml.BorderY("c"), vml.MarginL(1), vml.MarginY(2), vml.PadB(3)))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}
