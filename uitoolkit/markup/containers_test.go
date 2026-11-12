package markup

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// sameLayout fails unless the template and the Go view dump the same once
// both are laid out in a window of width by height.
func sameLayout(t *testing.T, ctx *core.Ctx, template, goView view.Node, width, height int) {
	t.Helper()
	view.Layout(ctx, template, width, height)
	view.Layout(ctx, goView, width, height)
	if got, want := view.Dump(template), view.Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

// buildError is the error Build gives for src and scope; src must parse.
func buildError(t *testing.T, src string, scope Scope) error {
	t.Helper()
	template, err := Parse("t.vml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = template.Build(scope)
	return err
}

func TestScrollElementsMatchTheGoScrollers(t *testing.T) {
	ctx := testCtx(t)
	column, strip := &widget.ScrollState{Y: 30}, &widget.ScrollState{}
	template := build(t, `
		<col>
		  <scroll state="column" h="100" gap="4"><row h="60"/><row h="60"/></scroll>
		  <hscroll state="strip" w="120" gap="2"><row w="80"/><row w="80" keep-in-view/></hscroll>
		</col>`, Scope{"column": column, "strip": strip})
	goView := view.Col(
		widget.Scroll(column, view.H(100), view.Gap(4), view.Row(view.H(60)), view.Row(view.H(60))),
		widget.HScroll(strip, view.W(120), view.Gap(2), view.Row(view.W(80)), view.Row(view.W(80), view.KeepInView())),
	)
	sameLayout(t, ctx, template, goView, 300, 200)
}

func TestScrollElementScrollsTheStateOfTheScope(t *testing.T) {
	ctx := testCtx(t)
	column := &widget.ScrollState{}
	template := build(t, `<col><scroll state="column" h="100"><row h="300"/></scroll></col>`, Scope{"column": column})
	view.Layout(ctx, template, 200, 100)
	ctx.In = core.Input{X: 50, Y: 50, Wheel: -1}
	view.Update(ctx, template)
	if column.Y <= 0 {
		t.Fatalf("a wheel turn down left the scope's state at Y = %d", column.Y)
	}
}

func TestKeepInViewScrollsTheStripToTheFlaggedChild(t *testing.T) {
	ctx := testCtx(t)
	strip := &widget.ScrollState{}
	template := build(t, `<row><hscroll state="strip" w="100"><row w="90"/><row w="90"/><row w="90" keep-in-view/></hscroll></row>`, Scope{"strip": strip})
	view.Layout(ctx, template, 300, 40)
	view.Update(ctx, template)
	if strip.X <= 0 {
		t.Fatalf("the strip stayed at X = %d, want it scrolled to the third child", strip.X)
	}
}

func TestScrollStateOfTheWrongTypeNamesTheTypeItWants(t *testing.T) {
	err := buildError(t, `<scroll state="column"/>`, Scope{"column": widget.ScrollState{}})
	if err == nil || !strings.Contains(err.Error(), "*widget.ScrollState") {
		t.Errorf("error = %v, want it to say column is not a *widget.ScrollState", err)
	}
}

func TestScrollWithoutStateIsAParseError(t *testing.T) {
	_, err := Parse("t.vml", []byte(`<scroll h="10"/>`))
	if err == nil || !strings.Contains(err.Error(), "state") {
		t.Errorf("error = %v, want it to say <scroll> needs state", err)
	}
}

func TestSplitElementMatchesTheGoSplit(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		goView     func(state *widget.SplitState, first, second view.Node) view.Node
	}{
		{"pixels", `default="200"`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 200, first, second)
		}},
		{"vertical", `default="150" vertical`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 150, widget.Vertical(), first, second)
		}},
		{"ratio", `default="0.25" as-ratio`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 0.25, widget.AsRatio(), first, second)
		}},
		{"first minimum", `default="10" min-first="120"`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 10, widget.MinFirst(120), first, second)
		}},
		{"second minimum", `default="590" min-second="150"`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 590, widget.MinSecond(150), first, second)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &widget.SplitState{}
			template := build(t, `<split state="split" `+test.attributes+`><row bg="panel"/><row bg="frame"/></split>`, Scope{"split": state})
			goView := test.goView(state, view.Row(view.Bg(core.Panel)), view.Row(view.Bg(core.Frame)))
			sameLayout(t, testCtx(t), template, goView, 601, 400)
		})
	}
}

func TestSplitElementDragMovesTheStateOfTheScope(t *testing.T) {
	ctx := testCtx(t)
	state := &widget.SplitState{}
	template := build(t, `<split state="split" default="200"><row/><row/></split>`, Scope{"split": state})
	view.Layout(ctx, template, 600, 100)
	ctx.In = core.Input{X: 200, Y: 40, Click: true, Held: true}
	view.Update(ctx, template)
	ctx.In = core.Input{X: 320, Y: 40, Held: true}
	view.Update(ctx, template)
	if state.Size != 320 {
		t.Fatalf("after dragging the divider to x=320 the scope's state holds %v", state.Size)
	}
}

func TestTabsElementsMatchTheGoTabs(t *testing.T) {
	tabs := []widget.TabItem{{Label: "Body"}, {Label: "Headers", Badge: 3}, {Label: "Settings"}}
	strip := &widget.ScrollState{}
	choose := func(int) {}
	template := build(t, `
		<col>
		  <tabs items="tabs" active="1" on-select="choose" h="30"/>
		  <scroll-tabs state="strip" items="tabs" active="{{active}}" w="100"/>
		</col>`, Scope{"tabs": tabs, "strip": strip, "choose": choose, "active": 2})
	goView := view.Col(
		widget.Tabs(tabs, 1, view.OnSelect(choose), view.H(30)),
		widget.ScrollTabs(strip, tabs, 2, view.W(100)),
	)
	sameLayout(t, testCtx(t), template, goView, 400, 100)
}

func TestTabsElementClickSelectsThroughTheScopeHandler(t *testing.T) {
	ctx := testCtx(t)
	chosen := -1
	template := build(t, `<col><tabs items="tabs" active="0" on-select="choose"/></col>`,
		Scope{"tabs": []widget.TabItem{{Label: "One"}, {Label: "Two"}}, "choose": func(index int) { chosen = index }})
	view.Layout(ctx, template, 400, 100)
	second := template.Children()[0].Children()[1].Rect()
	ctx.In = core.Input{X: second.X + second.W/2, Y: second.Y + second.H/2, Click: true}
	view.Update(ctx, template)
	if chosen != 1 {
		t.Fatalf("clicking the second tab chose %d, want 1", chosen)
	}
}

func TestSeparatorAndActiveTabAttributesMatchTheGoOptions(t *testing.T) {
	got := view.Dump(build(t, `<row separator="12"><row active-tab/></row>`, nil))
	want := view.Dump(view.Row(view.Separator(12), view.Row(widget.ActiveTab())))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

func TestTabsTakeNeitherChildrenNorText(t *testing.T) {
	for _, src := range []string{`<tabs items="x" active="0"><row/></tabs>`, `<tabs items="x" active="0">label</tabs>`} {
		if _, err := Parse("t.vml", []byte(src)); err == nil {
			t.Errorf("%s parsed", src)
		}
	}
}

// Split's options set the split behavior, so another element must not take them.
func TestSplitOptionsAreOnlyForSplit(t *testing.T) {
	for _, attribute := range []string{"vertical", "as-ratio", `min-first="1"`, `min-second="1"`, `default="1"`} {
		if _, err := Parse("t.vml", []byte(`<row `+attribute+`/>`)); err == nil {
			t.Errorf("<row %s> parsed", attribute)
		}
	}
}
