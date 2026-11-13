package markup

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func build(t *testing.T, src string, sc Scope) view.Node {
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
	c, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	clicked := false
	goView := view.Row(view.H(28), view.PadX(12), view.Gap(12), view.Bg(core.Frame), view.Control("statusbar"), view.BorderT(core.Border),
		view.Text("Ready", view.Small(), view.Muted(), view.Ellipsis(), view.Grow(1)),
		view.Text("v0.1.0", view.Small(), view.Muted(), view.OnClick(func() { clicked = true })),
		widget.Button("Go", view.W(60), view.Tooltip("Go on")),
	)
	tplView := build(t, `
		<row h="28" pad="12 0" gap="12" bg="frame" control="statusbar" border-t="border">
		  <text small muted ellipsis grow="1">{{status}}</text>
		  <text small muted on-click="bump">{{version}}</text>
		  <button w="60" tooltip="Go on">Go</button>
		</row>`, Scope{"status": "Ready", "version": "v0.1.0", "bump": func() { clicked = true }})
	view.Layout(c, goView, 600, 28)
	view.Layout(c, tplView, 600, 28)
	if g, w := view.Dump(tplView), view.Dump(goView); g != w {
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
	on := view.Dump(build(t, src, Scope{"show": true, "n": 3, "items": []map[string]any{{"name": "a"}, {"name": "b"}}}))
	for _, want := range []string{`text="shown 3"`, `text="0:a"`, `text="1:b"`} {
		if !strings.Contains(on, want) {
			t.Errorf("missing %s in\n%s", want, on)
		}
	}
	if strings.Contains(on, "hidden") {
		t.Errorf("if=!show built when show is true:\n%s", on)
	}
	off := view.Dump(build(t, src, Scope{"show": false, "n": 0, "items": []string{}}))
	if strings.Contains(off, "shown") || !strings.Contains(off, `text="hidden"`) || strings.Contains(off, "0:") {
		t.Errorf("show=false:\n%s", off)
	}
}

func TestAttributesCanBeInterpolated(t *testing.T) {
	n := build(t, `<row bg="{{tone}}" w="{{width}}"><text color="{{tone}}">x</text></row>`, Scope{"tone": "panel", "width": 40})
	d := view.Dump(n)
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
		"not a handler": {`<text on-click="x">a</text>`, Scope{"x": "str"}, "not a func()"},
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
	d := view.Dump(build(t, `<text small="false" bold="true">a&nbsp;b &amp; c</text>`, nil))
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
	got := view.Dump(bar.MustBuild(Scope{"status": "Ready"}))
	want := view.Dump(view.Row(view.H(28), view.Text("Ready", view.Small())))
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
	if !strings.Contains(view.Dump(a), `"1"`) || !strings.Contains(view.Dump(b), `"2"`) {
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
	_, err = Parse("t.vml", []byte(`<row border_t="border"/>`))
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
	for n, spec := range elements {
		if !strings.Contains(ref, "<"+n+">") {
			t.Errorf("reference misses element %s", n)
		}
		for _, name := range spec.ownNames() {
			if !strings.Contains(ref, " "+name) {
				t.Errorf("reference misses %s of <%s>", name, n)
			}
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
	for n := range handlers {
		if !strings.Contains(ref, "  "+n+"\n") {
			t.Errorf("reference misses handler %s", n)
		}
	}
}

// Template attributes are the Go options in kebab-case, for every side.
func TestSideAttributesMatchTheGoOptions(t *testing.T) {
	got := view.Dump(build(t, `<row border-r="a" border-x="b" border-y="c" margin-l="1" margin-y="2" pad-b="3"/>`, nil))
	want := view.Dump(view.Row(view.BorderR("a"), view.BorderX("b"), view.BorderY("c"), view.MarginL(1), view.MarginY(2), view.PadB(3)))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

func TestRemainingAttributesMatchTheGoOptions(t *testing.T) {
	got := view.Dump(build(t, `<stack hover-bg="hover" border-inset="3" float-tr="4 5" edge-t="accent-line" edge-l="accent-line"/>`, nil))
	want := view.Dump(view.Stack(view.HoverBg("hover"), view.BorderInset(3), view.FloatTR(4, 5), view.EdgeT("accent-line"), view.EdgeL("accent-line")))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
	for _, bad := range []string{`<row edge-t="x" float-tr="1"/>`, `<row float-at="1"/>`} {
		tp, err := Parse("t.vml", []byte(bad))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tp.Build(nil); err == nil {
			t.Errorf("%s built", bad)
		}
	}
}

func TestIconElementMatchesTheGoIcon(t *testing.T) {
	got := view.Dump(build(t, `<row><icon name="trash" w="20"/><icon name="{{which}}"/></row>`, Scope{"which": "plus"}))
	want := view.Dump(view.Row(widget.Icon("trash", view.W(20)), widget.Icon("plus")))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
	tp, err := Parse("t.vml", []byte(`<icon name="nope"/>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tp.Build(nil); err == nil || !strings.Contains(err.Error(), `no icon "nope"`) {
		t.Fatalf("err = %v", err)
	}
	if _, err := Parse("t.vml", []byte(`<row name="x"/>`)); err == nil {
		t.Fatal("name is only for icon")
	}
}

func TestSlotInsertsANodeFromTheScope(t *testing.T) {
	got := view.Dump(build(t, `<row><slot name="x"/><text>b</text></row>`, Scope{"x": view.Text("a")}))
	want := view.Dump(view.Row(view.Text("a"), view.Text("b")))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
	for _, sc := range []Scope{{}, {"x": "text"}} {
		tp, err := Parse("t.vml", []byte(`<slot name="x"/>`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tp.Build(sc); err == nil {
			t.Errorf("%v built", sc)
		}
	}
}

func TestAlignmentAttributesAndButtonsMatchTheGoOptions(t *testing.T) {
	got := view.Dump(build(t, `<row align-items="center" justify-content="between"><primary-button w="70">Go</primary-button><link-button>More</link-button></row>`, nil))
	want := view.Dump(view.Row(view.AlignItems(view.AlignCenter), view.JustifyContent(view.JustifyBetween), widget.PrimaryButton("Go", view.W(70)), widget.LinkButton("More")))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
	tp, err := Parse("t.vml", []byte(`<row align-items="middle"/>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tp.Build(nil); err == nil || !strings.Contains(err.Error(), "baseline, center") {
		t.Fatalf("err = %v", err)
	}
}

func TestIncludeBuildsAnotherDefinedTemplateWithTheSameScope(t *testing.T) {
	Define("test-badge", `<text small>{{label}}!</text>`)
	got := view.Dump(build(t, `<row><include name="test-badge"/><text for="x in xs">{{x}}</text></row>`, Scope{"label": "hi", "xs": []string{"a"}}))
	want := view.Dump(view.Row(view.Text("hi!", view.Small()), view.Text("a")))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
	tp, _ := Parse("t.vml", []byte(`<include name="nope"/>`))
	if _, err := tp.Build(nil); err == nil || !strings.Contains(err.Error(), "test-badge") {
		t.Fatalf("an unknown include lists the defined names: %v", err)
	}
	if _, err := Parse("t.vml", []byte(`<include/>`)); err != nil {
		t.Fatal(err)
	}
	tp2, _ := Parse("t.vml", []byte(`<include/>`))
	if _, err := tp2.Build(nil); err == nil {
		t.Fatal("an include without a name must fail")
	}
}

func TestIconHoverColorAttributeMirrorsTheGoOption(t *testing.T) {
	got := view.Dump(build(t, `<icon name="dots" color="hero-muted" hover-color="hero-text"/>`, nil))
	want := view.Dump(widget.Icon("dots", view.Color("hero-muted"), view.HoverColor("hero-text")))
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestLightenAttributeMatchesTheGoOption(t *testing.T) {
	got := view.Dump(build(t, `<row lighten/>`, nil))
	want := view.Dump(view.Row(view.Lighten()))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

func TestSizeLimitsAndFlagsAreAttributes(t *testing.T) {
	got := view.Dump(build(t, `<row min-w="10" max-w="20" min-h="5" max-h="9" shadow clip/>`, nil))
	want := view.Dump(view.Row(view.MinW(10), view.MaxW(20), view.MinH(5), view.MaxH(9), view.Shadow(), view.Clip()))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

// A view whose source is built from data (a name in a string) would add a
// cache entry per value for ever.
func TestViewCacheStaysBounded(t *testing.T) {
	for i := range maxViewCache + 50 {
		View(fmt.Sprintf(`<text>n%d</text>`, i), Scope{})
	}
	viewMu.Lock()
	n := len(viewCache)
	viewMu.Unlock()
	if n > maxViewCache {
		t.Fatalf("%d cached sources, want at most %d", n, maxViewCache)
	}
}

// An empty color token switches the option off, so one template serves
// an item with the option and one without (a drop target's outline).
func TestEmptyColorTokenAppliesNoOption(t *testing.T) {
	for _, attribute := range []string{"bg", "border", "border-t", "border-r", "border-b", "border-l", "border-x", "border-y", "edge-t", "edge-l", "edge-r", "hover-bg", "hover-color", "color"} {
		t.Run(attribute, func(t *testing.T) {
			got := view.Dump(build(t, `<row `+attribute+`="{{token}}"/>`, Scope{"token": core.Token("")}))
			if want := view.Dump(view.Row()); got != want {
				t.Fatalf("template:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestNonEmptyColorTokenStillAppliesItsOption(t *testing.T) {
	got := view.Dump(build(t, `<row border="{{token}}"/>`, Scope{"token": core.AccentLine}))
	if want := view.Dump(view.Row(view.Border(core.AccentLine))); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
	if got == view.Dump(view.Row()) {
		t.Fatal("Dump does not show the border, so the empty-token test proves nothing")
	}
}
