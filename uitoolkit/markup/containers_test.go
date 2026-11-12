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
